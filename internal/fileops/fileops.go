package fileops

import (
	"context"
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
	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/pterm/pterm"
)

// ErrTimestampPreserve is returned when a cross-device copy succeeded but the
// original timestamps could not be restored. The file was moved successfully.
var ErrTimestampPreserve = errors.New("could not preserve file timestamps")

// MoveContext carries the dependencies needed by file-move operations.
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

// MoveRequest holds the operation-specific parameters for a MoveFiles call.
type MoveRequest struct {
	Category  *models.Category
	Files     []os.DirEntry
	Extension string
	BatchID   string
	SourceDir string // actual directory of the files; may differ from Category.Source.Path when recursive
}

// MoveFiles processes files matching the given extension in req.SourceDir.
// Returns the names of files that were successfully processed.
func MoveFiles(ctx context.Context, mctx MoveContext, req MoveRequest) []string {
	category := req.Category
	files := req.Files
	var moved []string
	for _, file := range files {
		select {
		case <-ctx.Done():
			return moved
		default:
		}
		if !hasExtension(file, req.Extension) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			mctx.Logger.Error("failed to stat file", mctx.Logger.Args("file", file.Name(), "error", err.Error()))
			continue
		}

		sourcePath := filepath.Join(req.SourceDir, file.Name())

		destDir := category.Destination.Path
		tctx := tokens.TokenContext{Info: info, CategoryName: category.Name, Now: time.Now(), SourcePath: sourcePath}
		if template := category.Destination.OrganizeBy; template != "" {
			if subdir := tokens.ResolveGroupBy(template, tctx); subdir != "" {
				destDir = filepath.Join(category.Destination.Path, subdir)
			}
		}

		if err := CreateDirectory(destDir); err != nil {
			mctx.Logger.Error("failed to create directory", mctx.Logger.Args("path", destDir, "error", err.Error()))
			continue
		}

		tctx.DestDir = destDir
		destName := tokens.ResolveRename(category.Destination.Rename, tctx)
		destPath := filepath.Join(destDir, destName)

		strategy := category.Destination.ConflictStrategy
		if strategy == "" {
			strategy = "rename"
		}
		resolved, skip := applyConflictStrategy(mctx, strategy, ConflictArgs{
			Src:      sourcePath,
			Dst:      destPath,
			DestDir:  destDir,
			FileName: destName,
		})
		if skip {
			continue
		}
		destPath = resolved

		action := category.Destination.Action
		if err := dispatchAction(action, sourcePath, destPath); err != nil {
			if errors.Is(err, ErrTimestampPreserve) {
				mctx.Logger.Warn("file processed but timestamps could not be preserved", mctx.Logger.Args("file", sourcePath))
			} else {
				mctx.Logger.Warn("failed to perform action on file", mctx.Logger.Args("file", sourcePath, "action", action, "error", err.Error()))
				continue
			}
		}

		if mctx.History != nil {
			effectiveAction := action
			if effectiveAction == "" {
				effectiveAction = "move"
			}
			if err := mctx.History.Add(history.Entry{
				Source:      sourcePath,
				Destination: destPath,
				Timestamp:   time.Now(),
				BatchID:     req.BatchID,
				Action:      effectiveAction,
				Category:    category.Name,
			}); err != nil {
				mctx.Logger.Warn("failed to add to history", mctx.Logger.Args("error", err.Error()))
			}
		}

		mctx.Logger.Info("file processed", mctx.Logger.Args("action", action, "source", sourcePath, "destination", destPath))
		moved = append(moved, file.Name())
	}
	return moved
}

// FileAction executes a file operation from src to dst.
type FileAction interface {
	Execute(src, dst string) error
}

type moveAction struct{}
type copyAction struct{}
type symlinkAction struct{}

func (a *moveAction) Execute(src, dst string) error    { return moveFile(src, dst) }
func (a *copyAction) Execute(src, dst string) error    { return copyFile(src, dst) }
func (a *symlinkAction) Execute(src, dst string) error { return os.Symlink(src, dst) }

var fileActions = map[string]FileAction{
	"move":    &moveAction{},
	"copy":    &copyAction{},
	"symlink": &symlinkAction{},
}

// dispatchAction performs the file operation indicated by action.
// Supported values: "move" (default), "copy", "symlink".
func dispatchAction(action, src, dst string) error {
	fa, ok := fileActions[action]
	if !ok {
		fa = fileActions["move"]
	}
	return fa.Execute(src, dst)
}

// applyConflictStrategy checks whether destPath already exists and resolves the
// conflict according to strategy.
func applyConflictStrategy(ctx MoveContext, strategy string, args ConflictArgs) (resolved string, skip bool) {
	if _, err := os.Stat(args.Dst); err != nil {
		return args.Dst, false
	}
	resolver, ok := conflictResolvers[strategy]
	if !ok {
		resolver = conflictResolvers["rename"]
	}
	resolvedPath, shouldMove, err := resolver.Resolve(args)
	if err != nil {
		ctx.Logger.Error("failed to resolve conflict", ctx.Logger.Args("file", args.FileName, "error", err.Error()))
		return "", true
	}
	if !shouldMove {
		if msg := resolver.SkipMessage(); msg != "" {
			ctx.Logger.Info(msg, ctx.Logger.Args("file", args.FileName))
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

	if !isCrossDeviceError(err) {
		return err
	}

	copyErr := copyFile(src, dst)
	if copyErr != nil && !errors.Is(copyErr, ErrTimestampPreserve) {
		return fmt.Errorf("cross-device copy failed: %w", copyErr)
	}

	if err := os.Remove(src); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("cross-device move: copied to %s but could not remove source: %w", dst, err)
	}

	return copyErr
}

// isCrossDeviceError reports whether err is a rename failure caused by src and
// dst being on different filesystems or drives.
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}

	inner := linkErr.Err

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

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("%w: %w", ErrTimestampPreserve, err)
	}

	return nil
}

const maxConflictAttempts = 1000

// getUniqueDestinationPath ensures no file is overwritten by appending (n) if needed.
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

// hasExtension checks if a file has a given extension (case-insensitive).
// When extension is "all", every file matches.
func hasExtension(file os.DirEntry, extension string) bool {
	if strings.ToLower(extension) == "all" {
		return true
	}
	ext := "." + extension
	fileExt := strings.ToLower(filepath.Ext(file.Name()))
	return fileExt == strings.ToLower(ext)
}
