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

	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
)

// ErrTimestampPreserve is returned when a cross-device copy succeeded but the
// original timestamps could not be restored. The file was moved successfully.
var ErrTimestampPreserve = errors.New("could not preserve file timestamps")

// MoveContext carries the dependencies needed by file-move operations.
// History may be a *history.History (saved per file, used by watch mode) or a
// *history.Buffer (collected in memory and flushed once per batch by the
// one-shot run). Callers must leave it nil — not a typed-nil pointer — when
// history tracking is disabled.
type MoveContext struct {
	Logger  logger.Logger
	History history.Recorder
}

// CreateDirectory creates dir and all necessary parents with full permissions.
// It is idempotent: no error is returned when dir already exists.
func CreateDirectory(dir string) error {
	return os.MkdirAll(dir, 0o750)
}

// MoveRequest holds the operation-specific parameters for a MoveFiles call.
type MoveRequest struct {
	Category  *models.Category
	Files     []os.DirEntry
	Extension string
	BatchID   string
	SourceDir string // actual directory of the files; may differ from Category.Source.Path when recursive
	// LogEachMove logs an INFO line per processed file. Watch mode sets it to
	// report files as they arrive; batch mode leaves it false and logs a single
	// consolidated block in the caller instead.
	LogEachMove bool
}

// MoveResult holds the outcome of a MoveFiles call.
type MoveResult struct {
	Moved   []string      // names of files that were successfully processed
	Skipped int           // files skipped by conflict strategy (skip / hash_check duplicate)
	Bytes   int64         // total size of the successfully processed files
	Details []MovedDetail // source/destination of each processed file, in order
}

// MovedDetail records where a single processed file came from and went to.
type MovedDetail struct {
	Source      string
	Destination string
}

// MoveFiles processes files matching the given extension in req.SourceDir.
func MoveFiles(ctx context.Context, mctx MoveContext, req MoveRequest) MoveResult {
	category := req.Category
	files := req.Files
	var result MoveResult
	// One allocator per call seeds each destination directory once, then hands
	// out sequence numbers in memory instead of re-scanning the directory per file.
	seqAlloc := tokens.NewSeqAllocator()
	for _, file := range files {
		select {
		case <-ctx.Done():
			return result
		default:
		}
		if !filters.HasExtension(file, req.Extension) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			mctx.Logger.Error("failed to stat file", mctx.Logger.Args("file", file.Name(), "error", err.Error()))
			continue
		}

		sourcePath := filepath.Join(req.SourceDir, file.Name())

		tctx := tokens.TokenContext{Info: info, CategoryName: category.Name, Now: time.Now(), SourcePath: sourcePath, SeqAlloc: seqAlloc}
		destDir, destName := ResolveDestination(category, &tctx)

		if err := CreateDirectory(destDir); err != nil {
			mctx.Logger.Error("failed to create directory", mctx.Logger.Args("path", destDir, "error", err.Error()))
			continue
		}

		destPath := filepath.Join(destDir, destName)

		strategy := category.Destination.ConflictStrategy
		if strategy == "" {
			strategy = models.ConflictStrategyRename
		}
		action := category.Destination.Action
		if action == "" {
			action = models.ActionMove
		}
		resolved, skip, finalize, stratErr := applyConflictStrategy(mctx, strategy, ConflictArgs{
			Src:      sourcePath,
			Dst:      destPath,
			DestDir:  destDir,
			FileName: destName,
			Action:   action,
		})
		if stratErr != nil {
			mctx.Logger.Error("cannot process file", mctx.Logger.Args("file", sourcePath, "error", stratErr.Error()))
			continue
		}
		if skip {
			result.Skipped++
			continue
		}
		destPath = resolved

		actionErr := performAction(ctx, mctx, action, sourcePath, destPath, finalize)
		if actionErr != nil {
			if errors.Is(actionErr, ErrTimestampPreserve) {
				mctx.Logger.Warn("file processed but timestamps could not be preserved", mctx.Logger.Args("file", sourcePath))
			} else {
				mctx.Logger.Warn("failed to perform action on file", mctx.Logger.Args("file", sourcePath, "action", action, "destination", destPath, "conflict_strategy", strategy, "error", actionErr.Error()))
				continue
			}
		}

		if mctx.History != nil {
			if err := mctx.History.Add(history.Entry{
				Source:      sourcePath,
				Destination: destPath,
				Timestamp:   time.Now(),
				BatchID:     req.BatchID,
				Action:      string(action),
				Category:    category.Name,
			}); err != nil {
				mctx.Logger.Warn("failed to record history; undo will not work for this file",
					mctx.Logger.Args("file", sourcePath, "error", err.Error()))
			}
		}

		if req.LogEachMove {
			mctx.Logger.Info("file processed", mctx.Logger.Args("action", action, "source", sourcePath, "destination", destPath))
		}
		result.Details = append(result.Details, MovedDetail{Source: sourcePath, Destination: destPath})
		result.Moved = append(result.Moved, file.Name())
		result.Bytes += info.Size()
	}
	return result
}

// FileAction executes a file operation from src to dst.
type FileAction interface {
	Execute(ctx context.Context, src, dst string) error
}

type moveAction struct{}
type copyAction struct{}
type symlinkAction struct{}

func (a *moveAction) Execute(ctx context.Context, src, dst string) error {
	return MoveFileCtx(ctx, src, dst)
}

func (a *copyAction) Execute(ctx context.Context, src, dst string) error {
	return copyFile(ctx, src, dst)
}
func (a *symlinkAction) Execute(_ context.Context, src, dst string) error {
	if _, err := os.Lstat(src); err != nil {
		return fmt.Errorf("symlink source does not exist: %w", err)
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	return os.Symlink(absSrc, dst)
}

var fileActions = map[models.Action]FileAction{
	models.ActionMove:    &moveAction{},
	models.ActionCopy:    &copyAction{},
	models.ActionSymlink: &symlinkAction{},
}

// performAction runs the file action and then finalizes any destination that the
// conflict resolver set aside: on failure the original destination is restored,
// on success the set-aside copy is discarded. ErrTimestampPreserve counts as
// success (the file was placed; only timestamps could not be preserved). It
// returns the raw action error for the caller to log.
func performAction(ctx context.Context, mctx MoveContext, action models.Action, src, dst string, finalize FinalizeFunc) error {
	actionErr := dispatchAction(ctx, action, src, dst)
	if finalize != nil {
		failed := actionErr != nil && !errors.Is(actionErr, ErrTimestampPreserve)
		if ferr := finalize(failed); ferr != nil {
			mctx.Logger.Error("failed to finalize destination after conflict strategy",
				mctx.Logger.Args("file", dst, "error", ferr.Error()))
		}
	}
	return actionErr
}

// dispatchAction performs the file operation indicated by action.
// Supported values: ActionMove (default), ActionCopy, ActionSymlink.
func dispatchAction(ctx context.Context, action models.Action, src, dst string) error {
	fa, ok := fileActions[action]
	if !ok {
		return fmt.Errorf("unknown action %q", action)
	}
	return fa.Execute(ctx, src, dst)
}

// applyConflictStrategy checks whether destPath already exists and resolves the
// conflict according to strategy. Returns a non-nil error only for unknown strategies;
// resolver failures are logged internally and surfaced as skip=true, err=nil.
func applyConflictStrategy(ctx MoveContext, strategy models.ConflictStrategy, args ConflictArgs) (resolved string, skip bool, finalize FinalizeFunc, err error) {
	if _, err := os.Stat(args.Dst); err != nil {
		return args.Dst, false, nil, nil
	}
	resolver, ok := conflictResolvers[strategy]
	if !ok {
		return "", true, nil, fmt.Errorf("unknown conflict strategy %q", strategy)
	}
	resolvedPath, shouldMove, fin, resolveErr := resolver.Resolve(args)
	if resolveErr != nil {
		ctx.Logger.Error("failed to resolve conflict", ctx.Logger.Args("file", args.FileName, "error", resolveErr.Error()))
		return "", true, nil, nil
	}
	if !shouldMove {
		if msg := resolver.SkipMessage(args); msg != "" {
			ctx.Logger.Info(msg, ctx.Logger.Args("file", args.FileName))
		}
		return "", true, nil, nil
	}
	return resolvedPath, false, fin, nil
}

// MoveFileCtx attempts to move a file from source to destination.
// Falls back to copy+delete when os.Rename fails across different devices/drives.
func MoveFileCtx(ctx context.Context, src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	if !isCrossDeviceError(err) {
		return err
	}

	copyErr := copyFile(ctx, src, dst)
	if copyErr != nil && !errors.Is(copyErr, ErrTimestampPreserve) {
		return fmt.Errorf("cross-device copy failed: %w", copyErr)
	}

	if err := os.Remove(src); err != nil {
		if cleanupErr := os.Remove(dst); cleanupErr != nil {
			return fmt.Errorf("cross-device move: copied to %s, could not remove source (%w); cleanup of destination also failed (%s) — both copies exist", dst, err, cleanupErr)
		}
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

// ctxReader wraps an io.Reader and aborts reads when the context is cancelled.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *ctxReader) Read(p []byte) (int, error) {
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
		return cr.r.Read(p)
	}
}

// copyFile copies src to dst preserving the original file mode and timestamps.
func copyFile(ctx context.Context, src, dst string) (retErr error) {
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
	outClosed := false
	defer func() {
		if retErr != nil && !errors.Is(retErr, ErrTimestampPreserve) {
			if !outClosed {
				_ = out.Close()
			}
			_ = os.Remove(dst)
		}
	}()

	if _, err := io.Copy(out, &ctxReader{ctx: ctx, r: in}); err != nil {
		return err
	}

	if err := out.Sync(); err != nil {
		return err
	}

	outClosed = true
	if err := out.Close(); err != nil {
		return err
	}

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("%w: %w", ErrTimestampPreserve, err)
	}

	return nil
}

const maxConflictAttempts = 1000

// UniqueDestination returns a path in destDir for fileName that does not collide
// with an existing file, appending (n) before the extension when needed.
func UniqueDestination(destDir, fileName string) (string, error) {
	return getUniqueDestinationPath(destDir, fileName)
}

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

	return "", fmt.Errorf("could not find a unique destination for %q in %q after %d attempts", fileName, destDir, maxConflictAttempts)
}
