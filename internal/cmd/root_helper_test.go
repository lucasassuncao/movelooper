package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogExtensionResult_PlainWhenColorDisabled verifies that the scan summary
// message carries the category and count, and that with color disabled (as the
// JSON logger does) it contains no ANSI escape codes, keeping structured logs clean.
func TestLogExtensionResult_PlainWhenColorDisabled(t *testing.T) {
	// Not parallel: toggles pterm's global color state.
	pterm.DisableColor()
	t.Cleanup(func() { pterm.EnableColor() })

	var buf bytes.Buffer
	m := &models.Movelooper{Logger: logger.NewSlog(&buf, "info", false)}

	logExtensionResult(m, nil, "images", "jpg", "move", false)

	dir := t.TempDir()
	for _, name := range []string{"a.jpg", "b.jpg"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	logExtensionResult(m, entries, "images", "jpg", "move", false)
	logExtensionResult(m, entries, "backups", "jpg", "archive", false)

	out := buf.String()
	assert.NotContains(t, out, "\x1b", "JSON output must not contain ANSI color codes")
	assert.Contains(t, out, "[images]")
	assert.Contains(t, out, "No .jpg files found")
	assert.Contains(t, out, "2 .jpg files to move")
	assert.Contains(t, out, "2 .jpg files to archive", "archive categories use the archive verb")
}

// TestFileNoun covers the scan-summary subject for both real extensions and the
// "all" sentinel, where ".all" is meaningless and the noun must be singular/plural.
func TestFileNoun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ext   string
		count int
		want  string
	}{
		{"jpg", 0, ".jpg files"},
		{"jpg", 1, ".jpg file"},
		{"jpg", 5, ".jpg files"},
		{"all", 0, "files"},
		{"all", 1, "file"},
		{"all", 2, "files"},
		{"ALL", 1, "file"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, fileNoun(c.ext, c.count), "fileNoun(%q, %d)", c.ext, c.count)
	}
}

// newBufMovelooper builds a Movelooper whose logs are captured in buf, for
// asserting on the orchestration output of runMove.
func newBufMovelooper(t *testing.T, buf *bytes.Buffer, cats []*models.Category) *models.Movelooper {
	t.Helper()
	hist, err := history.NewHistory(filepath.Join(t.TempDir(), "history.json"), 10)
	require.NoError(t, err)
	return &models.Movelooper{
		Logger:     logger.NewSlog(buf, "info", false),
		Categories: cats,
		History:    hist,
	}
}

func moveTestCategory(name, srcDir, dstDir, organizeBy string, exts []string) *models.Category {
	enabled := true
	return &models.Category{
		Name:    name,
		Enabled: &enabled,
		Source:  models.CategorySource{Path: srcDir, Extensions: exts},
		Destination: models.CategoryDestination{
			Path:             dstDir,
			OrganizeBy:       organizeBy,
			ConflictStrategy: models.ConflictStrategyRename,
		},
	}
}

// TestRunMove_DryRunShowsDestinations verifies that --dry-run logs the resolved
// destination (including the organize-by subdirectory) for each matched file and
// moves nothing.
func TestRunMove_DryRunShowsDestinations(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("x"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "sorted/{ext}", []string{"jpg"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{DryRun: true}))

	out := buf.String()
	assert.Contains(t, out, "Would move")
	assert.Contains(t, out, "[images]", "planned move should be prefixed with the category name")
	assert.Contains(t, out, "photo.jpg")
	assert.Contains(t, out, "sorted", "destination should include the resolved organize-by subdir")

	// Nothing was actually moved.
	assert.FileExists(t, filepath.Join(srcDir, "photo.jpg"))
	assert.NoFileExists(t, filepath.Join(dstDir, "sorted", "jpg", "photo.jpg"))
	assert.Empty(t, m.History.GetAllBatches())
}

// TestRunMove_ShowFilesConsolidatesMoves verifies that a real move with
// --show-files logs a single "[category] Moved" block listing every moved file,
// rather than one line per file.
func TestRunMove_ShowFilesConsolidatesMoves(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.jpg"), []byte("y"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "", []string{"jpg"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{ShowFiles: true}))

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, "Moved"), "moved files should be reported in a single consolidated block")
	assert.Contains(t, out, "Moved 2 .jpg files", "header carries the count, extension, and plural noun")
	assert.Contains(t, out, "[images]", "moved block should be prefixed with the category name")
	assert.Contains(t, out, "a.jpg")
	assert.Contains(t, out, "b.jpg")
	assert.NotContains(t, out, "file processed", "batch mode should not log per-file in the fileops layer")

	assert.FileExists(t, filepath.Join(dstDir, "a.jpg"))
	assert.FileExists(t, filepath.Join(dstDir, "b.jpg"))
}

// TestRunMove_ShowFilesBlockPerExtension verifies that files are reported in one
// block per extension, each header carrying its own count and singular/plural noun.
func TestRunMove_ShowFilesBlockPerExtension(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.jpg"), []byte("y"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "c.png"), []byte("z"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "", []string{"jpg", "png"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{ShowFiles: true}))

	out := buf.String()
	assert.Equal(t, 2, strings.Count(out, "Moved"), "one block per extension")
	assert.Contains(t, out, "Moved 2 .jpg files", "plural noun for the two jpgs")
	assert.Contains(t, out, "Moved 1 .png file", "singular noun for the single png")
}

// TestRunMove_WithoutShowFilesOmitsFileList verifies that a real move without
// --show-files moves the files but does not emit any per-file listing.
func TestRunMove_WithoutShowFilesOmitsFileList(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "", []string{"jpg"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	out := buf.String()
	assert.NotContains(t, out, "Moved", "no consolidated file block without --show-files")
	assert.NotContains(t, out, "file processed")
	assert.FileExists(t, filepath.Join(dstDir, "a.jpg"), "files are still moved without --show-files")
}

// TestRestoreEntries_ConsolidatesRestoredBlock verifies that undo reports all
// restored files under a single "file restored" log entry, not one per file.
func TestRestoreEntries_ConsolidatesRestoredBlock(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.jpg"), []byte("y"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "", []string{"jpg"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	batches := m.History.GetAllBatches()
	require.Len(t, batches, 1)
	entries := m.History.GetBatch(batches[0].BatchID)

	buf.Reset()
	restored := restoreEntries(context.Background(), m, entries, false)
	require.Len(t, restored, 2)

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, "file(s) restored"), "restored files should be reported in a single block")
	assert.Equal(t, 2, strings.Count(out, "\"path\":"), "both restored paths belong to the same log entry")
	assert.Contains(t, out, "a.jpg")
	assert.Contains(t, out, "b.jpg")
}

// TestRunMove_DryRunLeavesSeqTokenLiteral verifies that seq/hash tokens are not
// resolved (no directory scan, no hashing) in a dry run: they stay literal.
func TestRunMove_DryRunLeavesSeqTokenLiteral(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("x"), 0o644))

	cat := moveTestCategory("images", srcDir, dstDir, "", []string{"jpg"})
	cat.Destination.Rename = "{seq}_{name}"
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{DryRun: true}))

	assert.Contains(t, buf.String(), "{seq}_photo", "seq token should remain a literal placeholder in dry-run")
}

// TestRunMove_CategoryFilter verifies that --category restricts the run to the
// named category and leaves the others untouched.
func TestRunMove_CategoryFilter(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstA := t.TempDir()
	dstB := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.png"), []byte("x"), 0o644))

	catA := moveTestCategory("jpgs", srcDir, dstA, "", []string{"jpg"})
	catB := moveTestCategory("pngs", srcDir, dstB, "", []string{"png"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{catA, catB})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{CategoryFilter: "pngs"}))

	// Only the png category ran.
	assert.NoFileExists(t, filepath.Join(srcDir, "b.png"))
	assert.FileExists(t, filepath.Join(dstB, "b.png"))
	assert.FileExists(t, filepath.Join(srcDir, "a.jpg"))
	assert.NoFileExists(t, filepath.Join(dstA, "a.jpg"))
}

// TestRunMove_DisabledCategory verifies that a disabled category is skipped by
// default and processed only with --include-disabled.
func TestRunMove_DisabledCategory(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (srcDir, dstDir string, m *models.Movelooper) {
		t.Helper()
		srcDir, dstDir = t.TempDir(), t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("x"), 0o644))
		cat := moveTestCategory("off", srcDir, dstDir, "", []string{"jpg"})
		disabled := false
		cat.Enabled = &disabled
		var buf bytes.Buffer
		return srcDir, dstDir, newBufMovelooper(t, &buf, []*models.Category{cat})
	}

	t.Run("skipped by default", func(t *testing.T) {
		t.Parallel()
		srcDir, dstDir, m := setup(t)
		require.NoError(t, runMove(context.Background(), m, MoveOptions{}))
		assert.FileExists(t, filepath.Join(srcDir, "a.jpg"))
		assert.NoFileExists(t, filepath.Join(dstDir, "a.jpg"))
	})

	t.Run("included with --include-disabled", func(t *testing.T) {
		t.Parallel()
		srcDir, dstDir, m := setup(t)
		require.NoError(t, runMove(context.Background(), m, MoveOptions{IncludeDisabled: true}))
		assert.NoFileExists(t, filepath.Join(srcDir, "a.jpg"))
		assert.FileExists(t, filepath.Join(dstDir, "a.jpg"))
	})
}

// TestProcessCategoryMove_ArchiveReportsMovedCountToAfterHook is a regression
// test: an action: archive category's after hook must see the number of
// archived files in ML_FILES_MOVED, not 0.
func TestProcessCategoryMove_ArchiveReportsMovedCountToAfterHook(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "out.txt")
	for _, n := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		require.NoError(t, os.WriteFile(filepath.Join(src, n), []byte("data"), 0o644))
	}

	// Mirror hooks.defaultShell's own decision: on Windows a SHELL env var (e.g.
	// git-bash) takes priority over cmd, so the command syntax must match
	// whichever shell RunHook will actually invoke.
	var run []string
	if runtime.GOOS == "windows" && os.Getenv("SHELL") == "" {
		run = []string{fmt.Sprintf("echo %%ML_FILES_MOVED%% > %s", outFile)}
	} else {
		run = []string{fmt.Sprintf(`echo "$ML_FILES_MOVED" > %s`, filepath.ToSlash(outFile))}
	}

	cat := archiveTestCategory(src, dst, &models.ArchiveConfig{Format: "zip", Name: "{category}"})
	cat.Hooks = &models.CategoryHooks{After: &models.CategoryHook{OnFailure: "abort", Run: run}}

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	require.NoError(t, processCategoryMove(context.Background(), m, cat, batch))

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Equal(t, "3", strings.TrimSpace(string(data)))
}

// TestRunMove_InterruptedRunRecordsWhatItMoved is the regression test for the
// worst failure this tool had: a run interrupted partway through left files
// moved with no history at all, so nothing could be undone.
//
// The invariant asserted here holds wherever the cancellation happens to land:
// every file that reached the destination is in the history.
func TestRunMove_InterruptedRunRecordsWhatItMoved(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	const fileCount = 200
	for i := range fileCount {
		name := filepath.Join(srcDir, string(rune('a'+i%26))+time.Now().Format("150405.000000000")+".jpg")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o600))
	}

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel as soon as the run has visibly started moving files.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if entries, err := os.ReadDir(dstDir); err == nil && len(entries) > 0 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	_ = runMove(ctx, m, MoveOptions{})

	moved, err := os.ReadDir(dstDir)
	require.NoError(t, err)

	var recorded int
	for _, b := range m.History.GetAllBatches() {
		recorded += b.Count
	}
	assert.Equal(t, len(moved), recorded, "every file that reached the destination must be undoable")
}

// TestRunMove_InterruptedBeforeStartIsReported checks that a run cancelled
// before it touches anything exits non-zero and says so, rather than reporting
// a clean success that a scheduled job would read as "everything was organized".
func TestRunMove_InterruptedBeforeStartIsReported(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("data"), 0o600))

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runMove(ctx, m, MoveOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interrupted")

	assert.FileExists(t, filepath.Join(srcDir, "a.jpg"), "nothing should have moved")
	assert.Empty(t, m.History.GetAllBatches())
}

// TestRunMove_DryRunLeavesConflictingFilesUntouched covers the dry-run conflict
// preview: it must report the collision without writing anything.
func TestRunMove_DryRunLeavesConflictingFilesUntouched(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "a.jpg"), []byte("old"), 0o600))

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{DryRun: true}))

	src, err := os.ReadFile(filepath.Join(srcDir, "a.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(src))
	dst, err := os.ReadFile(filepath.Join(dstDir, "a.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(dst))
	assert.Empty(t, m.History.GetAllBatches())
}

// TestAppendPlannedMoves_DetectsExistingDestination pins the conflict detection
// itself, independent of how it is logged.
func TestAppendPlannedMoves_DetectsExistingDestination(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.jpg"), []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "a.jpg"), []byte("old"), 0o600))

	m := buildIntegrationMovelooper(t, srcDir, dstDir, filepath.Join(t.TempDir(), "h.json"), []string{"jpg"})

	dirEntries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	matched := make([]scanner.FileEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		matched = append(matched, scanner.FileEntry{Dir: srcDir, Entry: e})
	}

	planned, conflicts := appendPlannedMoves(nil, m.Categories[0], matched)
	assert.Len(t, planned, 8, "two source/destination key-value pairs")
	require.Len(t, conflicts, 1)
	assert.Equal(t, filepath.Join(dstDir, "a.jpg"), conflicts[0])
}
