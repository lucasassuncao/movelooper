package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/hooks"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/pterm/pterm"
)

// movedSet tracks absolute paths that have already been moved in the current
// batch, preventing a file from being claimed by more than one category.
type movedSet map[string]bool

func (s movedSet) mark(dir, name string)     { s[filepath.Join(dir, name)] = true }
func (s movedSet) has(dir, name string) bool { return s[filepath.Join(dir, name)] }

// runStats accumulates totals across all categories for the end-of-run summary.
type runStats struct {
	totalFiles   int
	totalBytes   int64
	skipped      int // categories that errored out
	filesSkipped int // files skipped by a conflict strategy (skip / hash_check duplicate)
	failed       int
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
	// recorder buffers history entries in memory; runMove flushes them to disk
	// once at the end of the run. nil when history tracking is disabled.
	recorder history.Recorder
}

// hookAfterVars carries the post-move stats needed for "after" hook env vars.
type hookAfterVars struct {
	moved       int
	skipped     int
	failed      int
	batchID     string
	archivePath string
}

// runMove executes the default move operation across all configured categories.
func runMove(ctx context.Context, m *models.Movelooper, opts MoveOptions) error {
	names := ParseCategoryNames(opts.CategoryFilter)
	categories, err := FilterCategories(m.Categories, names, opts.IncludeDisabled, m.Logger)
	if err != nil {
		return err
	}

	var stats runStats
	var histBuf *history.Buffer
	batch := moveBatch{
		moved:     make(movedSet),
		batchID:   history.NewBatchID(),
		dryRun:    opts.DryRun,
		showFiles: opts.ShowFiles,
		stats:     &stats,
	}
	if m.History != nil {
		histBuf = &history.Buffer{}
		batch.recorder = histBuf
	}

	for _, category := range categories {
		if err := processCategoryMove(ctx, m, category, batch); err != nil {
			m.Logger.Error("failed to process category",
				m.Logger.Args("category", category.Name, "error", err.Error()))
			batch.stats.skipped++
		}
	}

	if histBuf != nil {
		if err := histBuf.Flush(m.History); err != nil {
			m.Logger.Warn("failed to record history; undo will not work for this run",
				m.Logger.Args("error", err.Error()))
		}
	}

	if opts.DryRun {
		m.Logger.Info("dry-run complete, no files were moved")
	} else {
		m.Logger.Info("run complete",
			m.Logger.Args("moved", stats.totalFiles, "size", formatBytes(stats.totalBytes), "files_skipped", stats.filesSkipped, "categories_skipped", stats.skipped))
	}

	// Surface failures through the exit code so scripts and cron can detect them.
	// The run is not aborted on failure, only reported here after it completes.
	if stats.skipped > 0 || stats.failed > 0 {
		return fmt.Errorf("run completed with failures: %d categories failed, %d files failed to move", stats.skipped, stats.failed)
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
		if after.archivePath != "" {
			env["ML_ARCHIVE_PATH"] = after.archivePath
		}
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

	// seen claims each file for the first extension in the list that matches it,
	// so a file is never counted or moved twice when "all" is listed alongside
	// specific extensions (the "all" pass would otherwise re-grab everything).
	seen := make(map[string]bool, len(allEntries))

	isArchive := category.Destination.Action == models.ActionArchive
	pendingVerb, pastVerb := actionVerbs(category.Destination.Action)
	var archiveFiles []scanner.FileEntry
	var archiveBytes int64

	var totalMoved, totalSkipped, totalFailed int
	for _, extension := range category.Source.Extensions {
		candidates := byExt[extension]
		if strings.EqualFold(extension, filters.ExtAll) {
			candidates = allEntries
		}
		matched, matchedBytes := matchExtensionFiles(m, category, candidates, extension, batch, seen)

		asDirEntries := make([]os.DirEntry, len(matched))
		for i, fe := range matched {
			asDirEntries[i] = fe.Entry
		}
		logExtensionResult(m, asDirEntries, category.Name, extension, pendingVerb, batch.showFiles)

		switch {
		case isArchive:
			archiveFiles = append(archiveFiles, matched...)
			archiveBytes += matchedBytes
		case batch.dryRun:
			previewExtensionMove(m, category, matched, extension, pendingVerb, batch)
		case len(matched) > 0:
			t := moveMatchedFiles(ctx, m, category, matched, extension, batch)
			totalMoved += t.moved
			totalSkipped += t.skipped
			totalFailed += t.failed
			// Only files that were actually processed count towards the run
			// summary; skipped and failed files are reported separately.
			batch.stats.totalFiles += t.moved
			batch.stats.totalBytes += t.bytes
			if batch.showFiles {
				header := fmt.Sprintf("%s %d %s", pastVerb, t.moved, fileNoun(extension, t.moved))
				logFileBlock(m, category.Name, header, appendMovedDetails(nil, t.details))
			}
		}
	}

	var archivePath string
	if isArchive {
		path, err := archiveCategory(ctx, m, category, archiveFiles, batch)
		if err != nil {
			// Propagated so runMove counts the category as failed and the run
			// exits non-zero — cron/scripts must be able to detect a failed archive.
			return fmt.Errorf("archive: %w", err)
		}
		archivePath = path
		// Archived files count towards the summary only when the archive was
		// actually written (not on dry-run or a conflict-strategy skip).
		if archivePath != "" {
			batch.stats.totalFiles += len(archiveFiles)
			batch.stats.totalBytes += archiveBytes
			// Also count towards the after-hook's ML_FILES_MOVED, so it reflects
			// the archived files instead of always reporting 0 for this action.
			totalMoved += len(archiveFiles)
		}
	}

	batch.stats.failed += totalFailed
	batch.stats.filesSkipped += totalSkipped

	if category.Hooks != nil && category.Hooks.After != nil {
		env := hookEnv(category, batch.dryRun, &hookAfterVars{
			moved:       totalMoved,
			skipped:     totalSkipped,
			failed:      totalFailed,
			batchID:     batch.batchID,
			archivePath: archivePath,
		})
		if err := hooks.RunHook(ctx, category.Hooks.After, hooks.HookContext{Log: m.Logger, Stdout: os.Stdout, Stderr: os.Stderr}, env); err != nil {
			return fmt.Errorf("after hook: %w", err)
		}
	}
	return nil
}

// matchExtensionFiles returns the candidates that pass the category filters
// for extension and were not claimed by an earlier extension pass, along with
// their total size. Matched files are recorded in seen so the "all" sentinel
// never re-grabs a file already taken by a specific extension.
func matchExtensionFiles(m *models.Movelooper, category *models.Category, candidates []scanner.FileEntry, extension string, batch moveBatch, seen map[string]bool) ([]scanner.FileEntry, int64) {
	matched := make([]scanner.FileEntry, 0, len(candidates))
	var bytes int64
	for _, fe := range candidates {
		full := filepath.Join(fe.Dir, fe.Entry.Name())
		if seen[full] {
			continue
		}
		info, err := matchesCategory(category, fe, batch.moved, extension)
		if err != nil {
			m.Logger.Warn("skipping file: could not read metadata", m.Logger.Args("file", fe.Entry.Name(), "error", err.Error()))
			continue
		}
		if info != nil {
			seen[full] = true
			matched = append(matched, fe)
			bytes += info.Size()
		}
	}
	return matched, bytes
}

// previewExtensionMove logs the dry-run preview for one extension and claims
// the files in the shared moved set, so a later category does not preview the
// same file — mirroring how a real run claims them.
func previewExtensionMove(m *models.Movelooper, category *models.Category, matched []scanner.FileEntry, extension, pendingVerb string, batch moveBatch) {
	for _, fe := range matched {
		batch.moved.mark(fe.Dir, fe.Entry.Name())
	}
	plannedArgs := appendPlannedMoves(nil, category, matched)
	header := fmt.Sprintf("Would %s %d %s", pendingVerb, len(matched), fileNoun(extension, len(matched)))
	logFileBlock(m, category.Name, header, plannedArgs)
}

// logFileBlock logs a single "[category] header" entry listing all
// source/destination pairs in args. It is a no-op when args is empty.
func logFileBlock(m *models.Movelooper, categoryName, header string, args []any) {
	if len(args) == 0 {
		return
	}
	label := pterm.Cyan(fmt.Sprintf("[%s]", categoryName))
	m.Logger.Info(fmt.Sprintf("%s %s", label, header), m.Logger.Args(args...))
}

// moveTotals aggregates the per-directory outcomes of moving one extension.
type moveTotals struct {
	moved, skipped, failed int
	bytes                  int64
	details                []fileops.MovedDetail
}

// moveMatchedFiles moves the matched files grouped by source directory and
// returns the aggregated counts and per-file source/destination details.
func moveMatchedFiles(ctx context.Context, m *models.Movelooper, category *models.Category, matched []scanner.FileEntry, extension string, batch moveBatch) moveTotals {
	var t moveTotals
	for dir, dirFiles := range groupByDir(matched) {
		req := fileops.MoveRequest{
			Category:  category,
			Files:     dirFiles,
			Extension: extension,
			BatchID:   batch.batchID,
			SourceDir: dir,
		}
		res := moveExtensionWithResult(ctx, m, req, batch)
		t.moved += len(res.Moved)
		t.skipped += res.Skipped
		t.failed += max(0, len(dirFiles)-len(res.Moved)-res.Skipped)
		t.bytes += res.Bytes
		t.details = append(t.details, res.Details...)
	}
	return t
}

// appendMovedDetails appends "source"/"destination" pairs for each moved file.
func appendMovedDetails(args []any, details []fileops.MovedDetail) []any {
	for _, d := range details {
		args = append(args, "source", d.Source, "destination", d.Destination)
	}
	return args
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
func moveExtensionWithResult(ctx context.Context, m *models.Movelooper, req fileops.MoveRequest, batch moveBatch) fileops.MoveResult {
	mctx := fileops.MoveContext{Logger: m.Logger, History: batch.recorder}
	result := fileops.MoveFiles(ctx, mctx, req)
	for _, name := range result.Moved {
		batch.moved.mark(req.SourceDir, name)
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
	if !filters.MatchesFilter(category.Source.Filter, filepath.Join(fe.Dir, fe.Entry.Name()), info) {
		return nil, nil
	}
	return info, nil
}

// actionVerbs returns the present-tense ("move") and past-tense ("Moved") labels
// for a destination action, used in scan summaries, dry-run previews, and the
// consolidated file block so the wording matches the operation.
func actionVerbs(action models.Action) (pending, past string) {
	switch action {
	case models.ActionCopy:
		return "copy", "Copied"
	case models.ActionSymlink:
		return "link", "Linked"
	case models.ActionArchive:
		return "archive", "Archived"
	default: // move and the empty (default) action
		return "move", "Moved"
	}
}

// logPending logs a "files pending" summary. On the pretty (pterm) renderer it
// uses WARN so pending categories stand out visually (and surface even when the
// level is raised to warn); on the structured JSON logger it stays at INFO, so
// normal operation does not show up as warnings in log aggregators.
func logPending(m *models.Movelooper, message string, args ...[]pterm.LoggerArgument) {
	if _, pretty := m.Logger.(*pterm.Logger); pretty {
		m.Logger.Warn(message, args...)
		return
	}
	m.Logger.Info(message, args...)
}

// logExtensionResult logs a summary of files found for an extension. verb is the
// pending action ("move", "copy", "link", "archive") so the message reads
// "N files to <verb>".
// The category and count are colorized via pterm; in JSON mode color is
// disabled (see ConfigureLogger), so the structured message stays plain.
func logExtensionResult(m *models.Movelooper, files []os.DirEntry, categoryName, extension, verb string, showFiles bool) {
	category := pterm.Cyan(fmt.Sprintf("[%s]", categoryName))
	count := len(files)
	if count == 0 {
		m.Logger.Info(fmt.Sprintf("%s %s %s found", category, pterm.Red("No"), fileNoun(extension, 0)))
		return
	}
	message := fmt.Sprintf("%s %s %s to %s", category, pterm.Green(fmt.Sprintf("%d", count)), fileNoun(extension, count), verb)
	if showFiles {
		logArgs := filters.GenerateLogArgs(files, extension)
		if len(logArgs) > 0 {
			logPending(m, message, m.Logger.Args(logArgs...))
			return
		}
	}
	logPending(m, message)
}

// appendPlannedMoves resolves the destination for each matched file and appends
// "source"/"destination" pairs to args, so all planned moves for a category can
// be logged as a single entry in dry-run mode.
func appendPlannedMoves(args []any, category *models.Category, matched []scanner.FileEntry) []any {
	for _, fe := range matched {
		if src, dst, ok := resolvePlannedMove(category, fe); ok {
			args = append(args, "source", src, "destination", dst)
		}
	}
	return args
}

// resolvePlannedMove reports where a file would land under the category's
// organize-by and rename templates, without creating directories or moving
// anything. It shares fileops.ResolveDestination with the real move; DryRun
// leaves seq and hash tokens as literal placeholders (resolved only at move
// time). ok is false when the file's metadata could not be read.
func resolvePlannedMove(category *models.Category, fe scanner.FileEntry) (source, dest string, ok bool) {
	info, err := fe.Entry.Info()
	if err != nil {
		return "", "", false
	}
	sourcePath := filepath.Join(fe.Dir, fe.Entry.Name())
	tctx := tokens.TokenContext{Info: info, CategoryName: category.Name, Now: time.Now(), SourcePath: sourcePath, DryRun: true}
	destDir, destName := fileops.ResolveDestination(category, &tctx)

	return sourcePath, filepath.Join(destDir, destName), true
}

// fileNoun renders the file-count subject for a scan summary, agreeing in number
// with count ("file" vs "files"). The "all" sentinel drops the ".all" label since
// it is not a real extension; real extensions keep their ".ext" prefix.
func fileNoun(extension string, count int) string {
	noun := "files"
	if count == 1 {
		noun = "file"
	}
	if strings.EqualFold(extension, filters.ExtAll) {
		return noun
	}
	return fmt.Sprintf(".%s %s", extension, noun)
}
