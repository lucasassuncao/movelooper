package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/hooks"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
)

// movedSet tracks absolute paths that have already been moved in the current
// batch, preventing a file from being claimed by more than one category.
type movedSet map[string]bool

func (s movedSet) mark(dir, name string)     { s[filepath.Join(dir, name)] = true }
func (s movedSet) has(dir, name string) bool { return s[filepath.Join(dir, name)] }

// runStats accumulates totals across all categories for the end-of-run summary.
type runStats struct {
	totalFiles int
	totalBytes int64
	skipped    int
}

// MoveOptions carries the CLI flags for the move command.
type MoveOptions struct {
	DryRun          bool
	ShowFiles       bool
	CategoryFilter  string
	IncludeDisabled bool
}

// moveBatch groups the mutable state shared across a single move run.
type moveBatch struct {
	moved     movedSet
	batchID   string
	dryRun    bool
	showFiles bool
	stats     *runStats
}

// hookAfterVars carries the post-move stats needed for "after" hook env vars.
type hookAfterVars struct {
	moved   int
	skipped int
	failed  int
	batchID string
}

// runMove executes the default move operation across all configured categories.
func runMove(ctx context.Context, m *models.Movelooper, opts MoveOptions) error {
	names := models.ParseCategoryNames(opts.CategoryFilter)
	categories, err := models.FilterCategories(m.Categories, names, opts.IncludeDisabled, m.Logger)
	if err != nil {
		return err
	}

	var stats runStats
	batch := moveBatch{
		moved:     make(movedSet),
		batchID:   history.NewBatchID(),
		dryRun:    opts.DryRun,
		showFiles: opts.ShowFiles,
		stats:     &stats,
	}

	for _, category := range categories {
		if err := processCategoryMove(ctx, m, category, batch); err != nil {
			m.Logger.Error("failed to process category",
				m.Logger.Args("category", category.Name, "error", err.Error()))
			batch.stats.skipped++
		}
	}

	if opts.DryRun {
		m.Logger.Info("dry-run complete, no files were moved",
			m.Logger.Args("matched", stats.totalFiles))
	} else {
		m.Logger.Info("run complete",
			m.Logger.Args("moved", stats.totalFiles, "size", formatBytes(stats.totalBytes), "categories_skipped", stats.skipped))
	}
	return nil
}

// hookEnv builds the environment variable map to inject into a hook process.
// afterVars is non-nil only for "after" hooks.
func hookEnv(category *models.Category, dryRun bool, after *hookAfterVars) map[string]string {
	action := category.Destination.Action
	if action == "" {
		action = models.ActionMove
	}
	dry := "false"
	if dryRun {
		dry = "true"
	}
	env := map[string]string{
		"ML_CATEGORY":    category.Name,
		"ML_SOURCE_PATH": category.Source.Path,
		"ML_DEST_PATH":   category.Destination.Path,
		"ML_DRY_RUN":     dry,
		"ML_ACTION":      string(action),
	}
	if after != nil {
		env["ML_FILES_MOVED"] = fmt.Sprintf("%d", after.moved)
		env["ML_FILES_SKIPPED"] = fmt.Sprintf("%d", after.skipped)
		env["ML_FILES_FAILED"] = fmt.Sprintf("%d", after.failed)
		env["ML_BATCH_ID"] = after.batchID
	}
	return env
}

// processCategoryMove handles all extensions for a single category.
func processCategoryMove(ctx context.Context, m *models.Movelooper, category *models.Category, batch moveBatch) error {
	if category.Hooks != nil && category.Hooks.Before != nil {
		env := hookEnv(category, batch.dryRun, nil)
		if err := hooks.RunHook(ctx, category.Hooks.Before, hooks.HookContext{Log: m.Logger, Stdout: os.Stdout, Stderr: os.Stderr}, env); err != nil {
			return fmt.Errorf("before hook: %w", err)
		}
	}

	autoExclude := []string{category.Destination.Path}
	allEntries, err := scanner.WalkSource(category.Source, autoExclude)
	if err != nil {
		return fmt.Errorf("scan %q: %w", category.Source.Path, err)
	}

	var totalMoved, totalSkipped, totalFailed int
	for _, extension := range category.Source.Extensions {
		var matched []scanner.FileEntry
		for _, fe := range allEntries {
			info, err := matchesCategory(category, fe.Entry, batch.moved, extension)
			if err != nil {
				m.Logger.Warn("skipping file: could not read metadata", m.Logger.Args("file", fe.Entry.Name(), "error", err.Error()))
				continue
			}
			if info != nil {
				matched = append(matched, fe)
				batch.stats.totalBytes += info.Size()
			}
		}

		asDirEntries := make([]os.DirEntry, len(matched))
		for i, fe := range matched {
			asDirEntries[i] = fe.Entry
		}
		logExtensionResult(m, asDirEntries, category.Name, extension, batch.showFiles)
		batch.stats.totalFiles += len(matched)

		if !batch.dryRun && len(matched) > 0 {
			byDir := groupByDir(matched)
			for dir, dirFiles := range byDir {
				req := fileops.MoveRequest{
					Category:  category,
					Files:     dirFiles,
					Extension: extension,
					BatchID:   batch.batchID,
					SourceDir: dir,
				}
				res := moveExtensionWithResult(ctx, m, req, batch.moved)
				totalMoved += len(res.Moved)
				totalSkipped += res.Skipped
				totalFailed += len(dirFiles) - len(res.Moved) - res.Skipped
			}
		}
	}

	if category.Hooks != nil && category.Hooks.After != nil {
		env := hookEnv(category, batch.dryRun, &hookAfterVars{
			moved:   totalMoved,
			skipped: totalSkipped,
			failed:  totalFailed,
			batchID: batch.batchID,
		})
		if err := hooks.RunHook(ctx, category.Hooks.After, hooks.HookContext{Log: m.Logger, Stdout: os.Stdout, Stderr: os.Stderr}, env); err != nil {
			m.Logger.Warn("after hook failed",
				m.Logger.Args("category", category.Name, "error", err.Error()))
		}
	}
	return nil
}

// groupByDir groups FileEntries by their containing directory.
func groupByDir(entries []scanner.FileEntry) map[string][]os.DirEntry {
	result := make(map[string][]os.DirEntry)
	for _, fe := range entries {
		result[fe.Dir] = append(result[fe.Dir], fe.Entry)
	}
	return result
}

// moveExtensionWithResult moves files described by req and returns the MoveResult.
func moveExtensionWithResult(ctx context.Context, m *models.Movelooper, req fileops.MoveRequest, moved movedSet) fileops.MoveResult {
	mctx := fileops.MoveContext{Logger: m.Logger, History: m.History}
	result := fileops.MoveFiles(ctx, mctx, req)
	for _, name := range result.Moved {
		moved.mark(req.SourceDir, name)
	}
	return result
}

// formatBytes converts a byte count to a human-readable string (e.g. "1.23 MB").
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	const prefixes = "KMGTPE"
	if exp >= len(prefixes) {
		exp = len(prefixes) - 1
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), prefixes[exp])
}

// matchesCategory returns the file's FileInfo when it passes all category filters,
// nil when it does not match, or an error if metadata could not be read.
func matchesCategory(category *models.Category, file os.DirEntry, moved movedSet, extension string) (os.FileInfo, error) {
	if moved.has(category.Source.Path, file.Name()) {
		return nil, nil
	}
	if !file.Type().IsRegular() || !filters.HasExtension(file, extension) {
		return nil, nil
	}
	info, err := file.Info()
	if err != nil {
		return nil, fmt.Errorf("could not read metadata for %q: %w", file.Name(), err)
	}
	if !filters.MatchesFilter(category.Source.Filter, file.Name(), info) {
		return nil, nil
	}
	return info, nil
}

// logExtensionResult logs a summary of files found for an extension.
func logExtensionResult(m *models.Movelooper, files []os.DirEntry, categoryName, extension string, showFiles bool) {
	count := len(files)
	if count == 0 {
		m.Logger.Info(fmt.Sprintf("[%s] No .%s files found", categoryName, extension))
		return
	}
	message := fmt.Sprintf("[%s] %d .%s files to move", categoryName, count, extension)
	if showFiles {
		logArgs := filters.GenerateLogArgs(files, extension)
		if len(logArgs) > 0 {
			m.Logger.Warn(message, m.Logger.Args(logArgs...))
			return
		}
	}
	m.Logger.Warn(message)
}
