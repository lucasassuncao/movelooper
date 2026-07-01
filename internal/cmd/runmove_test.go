package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	restored := restoreEntries(context.Background(), m, entries)
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
