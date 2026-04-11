package helper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
)

// ErrTimestampPreserve is returned when a cross-device copy succeeded but the
// original timestamps could not be restored. The file was moved successfully.
var ErrTimestampPreserve = errors.New("could not preserve file timestamps")

// MoveContext carries the dependencies needed by file-move operations.
// It is intentionally narrow: callers supply only Logger and History,
// not the full Movelooper application object.
type MoveContext struct {
	Logger  *pterm.Logger
	History *history.History
}

// CreateDirectory creates dir and all necessary parents with full permissions.
// It is idempotent: no error is returned when dir already exists.
func CreateDirectory(dir string) error {
	return os.MkdirAll(dir, 0750)
}

// ReadDirectory reads the contents of a given directory and returns the files.
func ReadDirectory(path string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	return files, nil
}

// MoveFiles moves files with the specified extension from the source directory to the destination directory.
// When organize-by is set, files land in subdirectories resolved from the template; otherwise directly in <destination>/.
// Returns the names of files that were successfully moved.
func MoveFiles(ctx MoveContext, category *models.Category, files []os.DirEntry, extension, batchID string) []string {
	var moved []string
	for _, file := range files {
		if !HasExtension(file, extension) {
			continue
		}

		sourcePath := filepath.Join(category.Source.Path, file.Name())
		destDir := category.Destination.Path
		if template := category.Destination.OrganizeBy; template != "" {
			if info, err := file.Info(); err == nil {
				if subdir := ResolveGroupBy(template, info, category.Name, time.Now()); subdir != "" {
					destDir = filepath.Join(category.Destination.Path, subdir)
				}
			}
		}

		if err := CreateDirectory(destDir); err != nil {
			ctx.Logger.Error("failed to create directory", ctx.Logger.Args("path", destDir, "error", err.Error()))
			continue
		}

		destPath := filepath.Join(destDir, file.Name())

		strategy := category.Destination.ConflictStrategy
		if strategy == "" {
			strategy = "rename"
		}
		resolved, skip := applyConflictStrategy(ctx, strategy, sourcePath, destPath, destDir, file.Name())
		if skip {
			continue
		}
		destPath = resolved
		err := moveFile(sourcePath, destPath)
		if err != nil {
			if errors.Is(err, ErrTimestampPreserve) {
				ctx.Logger.Warn("file moved but timestamps could not be preserved", ctx.Logger.Args("file", sourcePath))
			} else {
				ctx.Logger.Error("failed to move file", ctx.Logger.Args("file", sourcePath, "error", err.Error()))
				continue
			}
		}

		if ctx.History != nil {
			err := ctx.History.Add(history.Entry{
				Source:      sourcePath,
				Destination: destPath,
				Timestamp:   time.Now(),
				BatchID:     batchID,
			})
			if err != nil {
				ctx.Logger.Warn("failed to add to history", ctx.Logger.Args("error", err.Error()))
			}
		}

		ctx.Logger.Info("file moved", ctx.Logger.Args("source", sourcePath, "destination", destPath))
		moved = append(moved, file.Name())
	}
	return moved
}

// applyConflictStrategy checks whether destPath already exists and resolves the
// conflict according to strategy. It returns the final destination path and
// whether the file should be skipped entirely.
func applyConflictStrategy(ctx MoveContext, strategy, sourcePath, destPath, destDir, fileName string) (resolved string, skip bool) {
	if _, err := os.Stat(destPath); err != nil {
		// Destination does not exist — no conflict.
		return destPath, false
	}
	resolvedPath, shouldMove, err := resolveConflict(strategy, sourcePath, destPath, destDir, fileName)
	if err != nil {
		ctx.Logger.Error("failed to resolve conflict", ctx.Logger.Args("file", fileName, "error", err.Error()))
		return "", true
	}
	if !shouldMove {
		switch strategy {
		case "skip":
			ctx.Logger.Info("file skipped due to conflict strategy", ctx.Logger.Args("file", fileName))
		case "hash_check":
			ctx.Logger.Info("duplicate file removed from source", ctx.Logger.Args("file", fileName))
		case "newest":
			ctx.Logger.Info("file skipped — destination is newer", ctx.Logger.Args("file", fileName))
		case "oldest":
			ctx.Logger.Info("file skipped — destination is older", ctx.Logger.Args("file", fileName))
		case "larger":
			ctx.Logger.Info("file skipped — destination is larger", ctx.Logger.Args("file", fileName))
		case "smaller":
			ctx.Logger.Info("file skipped — destination is smaller", ctx.Logger.Args("file", fileName))
		}
		return "", true
	}
	return resolvedPath, false
}

// moveFile attempts to move a file from source to destination.
// Falls back to copy+delete when os.Rename fails across different devices/drives.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// os.Rename fails across different filesystems/drives (EXDEV on Unix,
	// ERROR_NOT_SAME_DEVICE on Windows). Fall back to copy+delete only for
	// that specific error — other errors (permissions, missing file) are returned as-is.
	if !isCrossDeviceError(err) {
		return err
	}

	copyErr := copyFile(src, dst)
	if copyErr != nil && !errors.Is(copyErr, ErrTimestampPreserve) {
		return fmt.Errorf("cross-device copy failed: %w", copyErr)
	}

	if err := os.Remove(src); err != nil {
		// Copy succeeded but source removal failed. Remove the destination copy
		// to avoid silent duplication, then surface the error.
		_ = os.Remove(dst)
		return fmt.Errorf("cross-device move: copied to %s but could not remove source: %w", dst, err)
	}

	// Propagate timestamp warning so the caller can log it.
	return copyErr
}

// isCrossDeviceError reports whether err is a rename failure caused by src and
// dst being on different filesystems or drives.
//
// On Unix the kernel returns EXDEV; on Windows it returns ERROR_NOT_SAME_DEVICE
// (errno 17). Both are wrapped inside *os.LinkError by os.Rename, so we unwrap
// to the inner syscall error before comparing — this avoids treating unrelated
// *os.LinkError values (e.g. permission denied) as cross-device errors.
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}

	inner := linkErr.Err

	// syscall.EXDEV is defined on all Unix-like platforms.
	// On Windows, syscall.Errno(17) is ERROR_NOT_SAME_DEVICE.
	const windowsErrorNotSameDevice = syscall.Errno(17)

	switch runtime.GOOS {
	case "windows":
		return errors.Is(inner, windowsErrorNotSameDevice)
	default:
		return errors.Is(inner, syscall.EXDEV)
	}
}

// copyFile copies src to dst preserving the original file mode and timestamps.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(filepath.Clean(src)) //#nosec G304 -- path comes from directory walk, validated by caller
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(filepath.Clean(dst), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode()) //#nosec G304 -- path comes from directory walk, validated by caller
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}

	// Restore the original modification time so watchers and filters
	// that use file age (min-age / max-age) behave consistently.
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("%w: %w", ErrTimestampPreserve, err)
	}

	return nil
}

const maxConflictAttempts = 1000

// getUniqueDestinationPath ensures no file is overwritten by appending (n) if needed.
// Returns an error if no free slot is found within maxConflictAttempts tries.
func getUniqueDestinationPath(destDir, fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	nameOnly := strings.TrimSuffix(fileName, ext)

	destPath := filepath.Join(destDir, fileName)
	for counter := 1; counter <= maxConflictAttempts; counter++ {
		if _, err := os.Stat(destPath); err != nil {
			return destPath, nil
		}
		newName := fmt.Sprintf("%s(%d)%s", nameOnly, counter, ext)
		destPath = filepath.Join(destDir, newName)
	}

	return "", fmt.Errorf("could not find a unique destination for %q after %d attempts", fileName, maxConflictAttempts)
}
