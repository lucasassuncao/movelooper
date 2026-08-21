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
	// Vanished counts source files that were gone by the time they were
	// processed — already handled by another process, not a failure.
	Vanished int
	// HistoryFailed counts files that were processed but whose history entry
	// could not be recorded, so they cannot be undone.
	HistoryFailed int
	// Fatal is set when the batch stopped early because the destination became
	// unusable (see ErrFatalDestination). Files after that point were not tried.
	Fatal error
}

// MovedDetail records where a single processed file came from and went to.
type MovedDetail struct {
	Source      string
	Destination string
}

// MoveFiles processes files matching the given extension in req.SourceDir.
func MoveFiles(ctx context.Context, mctx MoveContext, req MoveRequest) MoveResult {
	var result MoveResult
	// One allocator per call seeds each destination directory once, then hands
	// out sequence numbers in memory instead of re-scanning the directory per file.
	seqAlloc := tokens.NewSeqAllocator()
	// consecutiveFatal counts back-to-back unrecoverable destination failures;
	// any file that gets through resets it, so an isolated one does not stop
	// the batch.
	var consecutiveFatal int

	for _, file := range req.Files {
		select {
		case <-ctx.Done():
			return result
		default:
		}
		if !filters.HasExtension(file, req.Extension) {
			continue
		}

		res := moveOneFile(ctx, mctx, req, file, seqAlloc)
		if res.historyFailed {
			result.HistoryFailed++
		}

		switch res.outcome {
		case outcomeMoved:
			consecutiveFatal = 0
			result.Details = append(result.Details, res.detail)
			result.Moved = append(result.Moved, file.Name())
			result.Bytes += res.size
		case outcomeSkipped:
			consecutiveFatal = 0
			result.Skipped++
		case outcomeVanished:
			result.Vanished++
		case outcomeFatal:
			consecutiveFatal++
			if consecutiveFatal >= maxConsecutiveFatalErrors {
				result.Fatal = res.err
				return result
			}
		case outcomeFailed:
			// Already logged where it happened; carry on with the next file.
		}
	}
	return result
}

// fileOutcome is what happened to a single file inside MoveFiles.
type fileOutcome int

const (
	outcomeMoved    fileOutcome = iota // placed at the destination
	outcomeSkipped                     // the conflict strategy declined it
	outcomeVanished                    // the source was already gone
	outcomeFailed                      // failed for a reason specific to this file
	outcomeFatal                       // the destination itself is unusable
)

// fileResult carries one file's outcome back to the MoveFiles loop.
type fileResult struct {
	outcome       fileOutcome
	detail        MovedDetail
	size          int64
	historyFailed bool
	err           error // the fatal error, when outcome is outcomeFatal
}

// vanished builds the result for a source file that is no longer on disk. This
// is the expected outcome when another process moved it between the directory
// scan and now, so it is logged at debug level and never counted as a failure.
func vanished(mctx MoveContext, sourcePath string) fileResult {
	mctx.Logger.Debug("file is already gone, nothing to do", mctx.Logger.Args("file", sourcePath))
	return fileResult{outcome: outcomeVanished}
}

// moveOneFile resolves the destination for a single file, applies the conflict
// strategy, performs the action, and records the move in history.
func moveOneFile(ctx context.Context, mctx MoveContext, req MoveRequest, file os.DirEntry, seqAlloc *tokens.SeqAllocator) fileResult {
	category := req.Category
	sourcePath := filepath.Join(req.SourceDir, file.Name())

	info, err := file.Info()
	if err != nil {
		if sourceVanished(sourcePath, err) {
			return vanished(mctx, sourcePath)
		}
		mctx.Logger.Error("failed to stat file", mctx.Logger.Args("file", file.Name(), "error", err.Error()))
		return fileResult{outcome: outcomeFailed}
	}

	tctx := tokens.TokenContext{Info: info, CategoryName: category.Name, Now: time.Now(), SourcePath: sourcePath, SeqAlloc: seqAlloc}
	destDir, destName := ResolveDestination(category, &tctx)

	if err := CreateDirectory(destDir); err != nil {
		mctx.Logger.Error("failed to create directory", mctx.Logger.Args("path", destDir, "error", err.Error()))
		return destinationFailure(destDir, err)
	}

	strategy := effectiveStrategy(category)
	action := effectiveAction(category)
	destPath := filepath.Join(destDir, destName)

	resolved, skip, finalize, stratErr := applyConflictStrategy(mctx, strategy, ConflictArgs{
		Src:      sourcePath,
		Dst:      destPath,
		DestDir:  destDir,
		FileName: destName,
		Action:   action,
	})
	if stratErr != nil {
		if sourceVanished(sourcePath, stratErr) {
			return vanished(mctx, sourcePath)
		}
		mctx.Logger.Error("cannot process file", mctx.Logger.Args("file", sourcePath, "error", stratErr.Error()))
		return fileResult{outcome: outcomeFailed}
	}
	if skip {
		return fileResult{outcome: outcomeSkipped}
	}
	destPath = resolved

	if res, done := runFileAction(ctx, mctx, actionArgs{
		action:     action,
		strategy:   strategy,
		sourcePath: sourcePath,
		destDir:    destDir,
		destPath:   destPath,
		finalize:   finalize,
	}); done {
		return res
	}

	result := fileResult{
		outcome: outcomeMoved,
		detail:  MovedDetail{Source: sourcePath, Destination: destPath},
		size:    info.Size(),
	}
	result.historyFailed = recordMove(mctx, req, category.Name, string(action), sourcePath, destPath)

	if req.LogEachMove {
		mctx.Logger.Info("file processed", mctx.Logger.Args("action", action, "source", sourcePath, "destination", destPath))
	}
	return result
}

// actionArgs groups everything runFileAction needs to perform one operation and
// describe it if it goes wrong.
type actionArgs struct {
	action     models.Action
	strategy   models.ConflictStrategy
	sourcePath string
	destDir    string
	destPath   string
	finalize   FinalizeFunc
}

// runFileAction performs the file operation. done is true when the caller must
// stop and return res; when false the action succeeded and processing continues.
// A timestamp-preservation failure counts as success: the file was placed.
func runFileAction(ctx context.Context, mctx MoveContext, args actionArgs) (res fileResult, done bool) {
	actionErr := performAction(ctx, mctx, args.action, args.sourcePath, args.destPath, args.finalize)
	switch {
	case actionErr == nil:
		return fileResult{}, false
	case errors.Is(actionErr, ErrTimestampPreserve):
		mctx.Logger.Warn("file processed but timestamps could not be preserved", mctx.Logger.Args("file", args.sourcePath))
		return fileResult{}, false
	case sourceVanished(args.sourcePath, actionErr):
		return vanished(mctx, args.sourcePath), true
	default:
		mctx.Logger.Warn("failed to perform action on file",
			mctx.Logger.Args("file", args.sourcePath, "action", args.action, "destination", args.destPath,
				"conflict_strategy", args.strategy, "error", actionErr.Error()))
		return destinationFailure(args.destDir, actionErr), true
	}
}

// destinationFailure classifies a destination write failure: unrecoverable ones
// (a full disk, a directory we cannot write to) are reported as fatal so the
// caller can stop repeating the same error for every remaining file.
func destinationFailure(destDir string, err error) fileResult {
	if isFatalDestinationError(err) {
		return fileResult{outcome: outcomeFatal, err: fmt.Errorf("%w: %s: %w", ErrFatalDestination, destDir, err)}
	}
	return fileResult{outcome: outcomeFailed}
}

// recordMove adds the move to the history. It reports whether recording failed,
// which means this file cannot be undone — loud enough to log as an error, but
// never a reason to undo the move that already happened.
func recordMove(mctx MoveContext, req MoveRequest, categoryName, action, sourcePath, destPath string) (failed bool) {
	if mctx.History == nil {
		return false
	}
	err := mctx.History.Add(history.Entry{
		Source:      sourcePath,
		Destination: destPath,
		Timestamp:   time.Now(),
		BatchID:     req.BatchID,
		Action:      action,
		Category:    categoryName,
	})
	if err != nil {
		mctx.Logger.Error("failed to record history; this file cannot be undone",
			mctx.Logger.Args("file", sourcePath, "destination", destPath, "error", err.Error()))
		return true
	}
	return false
}

// effectiveStrategy returns the category's conflict strategy, defaulting to rename.
func effectiveStrategy(category *models.Category) models.ConflictStrategy {
	if s := category.Destination.ConflictStrategy; s != "" {
		return s
	}
	return models.ConflictStrategyRename
}

// effectiveAction returns the category's action, defaulting to move.
func effectiveAction(category *models.Category) models.Action {
	if a := category.Destination.Action; a != "" {
		return a
	}
	return models.ActionMove
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
	if _, statErr := os.Stat(args.Dst); statErr != nil {
		if !os.IsNotExist(statErr) {
			// Anything other than "does not exist" (e.g. permission denied) means we
			// cannot tell whether a conflict exists, so skip rather than risk an
			// unintended overwrite.
			ctx.Logger.Error("failed to check destination for conflicts",
				ctx.Logger.Args("file", args.FileName, "error", statErr.Error()))
			return "", true, nil, nil
		}
		return args.Dst, false, nil, nil
	}
	resolver, ok := conflictResolvers[strategy]
	if !ok {
		return "", true, nil, fmt.Errorf("unknown conflict strategy %q", strategy)
	}
	resolvedPath, shouldMove, fin, resolveErr := resolver.Resolve(args)
	if resolveErr != nil {
		// A source that disappeared mid-run is not a conflict failure; hand the
		// error back so the caller can count it as vanished rather than skipped.
		if sourceVanished(args.Src, resolveErr) {
			return "", false, nil, resolveErr
		}
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
