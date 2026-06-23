package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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
	assert.Contains(t, out, "would move")
	assert.Contains(t, out, "photo.jpg")
	assert.Contains(t, out, "sorted", "destination should include the resolved organize-by subdir")

	// Nothing was actually moved.
	assert.FileExists(t, filepath.Join(srcDir, "photo.jpg"))
	assert.NoFileExists(t, filepath.Join(dstDir, "sorted", "jpg", "photo.jpg"))
	assert.Empty(t, m.History.GetAllBatches())
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
