package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	names := ParseCategoryNames(opts.CategoryFilter)
	categories, err := FilterCategories(m.Categories, names, opts.IncludeDisabled, m.Logger)
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
		m.Logger.Info("dry-run complete, no files were moved")
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
	allEntries, err := scanner.WalkSource(ctx, category.Source, autoExclude)
	if err != nil {
		return fmt.Errorf("scan %q: %w", category.Source.Path, err)
	}

	// Group entries by extension in one pass to avoid O(n × extensions) re-scans.
	byExt := make(map[string][]scanner.FileEntry, len(category.Source.Extensions))
	for _, fe := range allEntries {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fe.Entry.Name()), "."))
		byExt[ext] = append(byExt[ext], fe)
	}

	var totalMoved, totalSkipped, totalFailed int
	for _, extension := range category.Source.Extensions {
		matched := make([]scanner.FileEntry, 0, len(byExt[extension]))
		for _, fe := range byExt[extension] {
			info, err := matchesCategory(category, fe, batch.moved, extension)
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
				totalFailed += max(0, len(dirFiles)-len(res.Moved)-res.Skipped)
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
			return fmt.Errorf("after hook: %w", err)
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
// Output is always in decimal units (1 KB = 1000 B), matching how ParseSize
// reads the suffixes without "i". Binary units (KiB/MiB/GiB) exist only on the
// input side: a size written as "1MiB" in the config is parsed to 1048576
// bytes and would be printed here as "1.05 MB" — same quantity, decimal label.
func formatBytes(b int64) string {
	if b < 1000 {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	val := float64(b) / 1000
	for _, u := range units {
		if val < 1000 || u == units[len(units)-1] {
			return fmt.Sprintf("%.2f %s", val, u)
		}
		val /= 1000
	}
	return "" // unreachable: the loop always returns at the last unit
}

// matchesCategory returns the file's FileInfo when it passes all category filters,
// nil when it does not match, or an error if metadata could not be read.
func matchesCategory(category *models.Category, fe scanner.FileEntry, moved movedSet, extension string) (os.FileInfo, error) {
	if moved.has(fe.Dir, fe.Entry.Name()) {
		return nil, nil
	}
	if !fe.Entry.Type().IsRegular() || !filters.HasExtension(fe.Entry, extension) {
		return nil, nil
	}
	info, err := fe.Entry.Info()
	if err != nil {
		return nil, fmt.Errorf("could not read metadata for %q: %w", fe.Entry.Name(), err)
	}
	if !filters.MatchesFilter(category.Source.Filter, fe.Entry.Name(), info) {
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
			m.Logger.Info(message, m.Logger.Args(logArgs...))
			return
		}
	}
	m.Logger.Info(message)
}
