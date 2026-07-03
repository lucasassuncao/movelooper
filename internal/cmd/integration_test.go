package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIntegrationLogger() *pterm.Logger {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &l
}

func buildIntegrationMovelooper(t *testing.T, srcDir, dstDir, histPath string, extensions []string) *models.Movelooper {
	t.Helper()
	enabled := true
	hist, err := history.NewHistory(histPath, 10)
	require.NoError(t, err)
	return &models.Movelooper{
		Logger: newIntegrationLogger(),
		Categories: []*models.Category{
			{
				Name:    "integration",
				Enabled: &enabled,
				Source: models.CategorySource{
					Path:       srcDir,
					Extensions: extensions,
				},
				Destination: models.CategoryDestination{
					Path:             dstDir,
					ConflictStrategy: models.ConflictStrategyRename,
				},
			},
		},
		History: hist,
	}
}

// TestIntegration_MoveThenUndo covers the full cycle: files are moved to the
// destination, then restored via undo, leaving the source directory as it was.
func TestIntegration_MoveThenUndo(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	for _, name := range []string{"a.jpg", "b.jpg", "readme.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, name), []byte("data"), 0o644))
	}

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})

	// --- move ---
	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	assert.FileExists(t, filepath.Join(dstDir, "a.jpg"))
	assert.FileExists(t, filepath.Join(dstDir, "b.jpg"))
	assert.NoFileExists(t, filepath.Join(srcDir, "a.jpg"))
	assert.NoFileExists(t, filepath.Join(srcDir, "b.jpg"))
	assert.FileExists(t, filepath.Join(srcDir, "readme.txt")) // not in extensions

	batches := m.History.GetAllBatches()
	require.Len(t, batches, 1)
	assert.Equal(t, 2, batches[0].Count)

	// --- undo ---
	entries := m.History.GetBatch(batches[0].BatchID)
	restored := restoreEntries(context.Background(), m, entries)
	require.Len(t, restored, 2)

	assert.FileExists(t, filepath.Join(srcDir, "a.jpg"))
	assert.FileExists(t, filepath.Join(srcDir, "b.jpg"))
	assert.NoFileExists(t, filepath.Join(dstDir, "a.jpg"))
	assert.NoFileExists(t, filepath.Join(dstDir, "b.jpg"))
}

// TestIntegration_AllExtensionMovesEverything verifies that the "all" sentinel
// in source.extensions matches files of any extension in the one-shot run,
// mirroring the behavior already honored by watch mode.
func TestIntegration_AllExtensionMovesEverything(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	names := []string{"a.jpg", "notes.txt", "archive.zip", "noext"}
	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, name), []byte("data"), 0o644))
	}

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"all"})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	for _, name := range names {
		assert.FileExists(t, filepath.Join(dstDir, name))
		assert.NoFileExists(t, filepath.Join(srcDir, name))
	}

	batches := m.History.GetAllBatches()
	require.Len(t, batches, 1)
	assert.Equal(t, len(names), batches[0].Count)
}

// TestIntegration_FailingCategoryReturnsError verifies that a category that
// cannot be processed (here: a non-existent source directory) makes runMove
// return an error, so the process exits non-zero for scripts and cron.
func TestIntegration_FailingCategoryReturnsError(t *testing.T) {
	t.Parallel()

	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")
	missingSrc := filepath.Join(t.TempDir(), "does-not-exist")

	m := buildIntegrationMovelooper(t, missingSrc, dstDir, histPath, []string{"jpg"})

	err := runMove(context.Background(), m, MoveOptions{})
	require.Error(t, err)
}

// TestIntegration_DryRunMovesNothing verifies that --dry-run reports files
// without touching them.
func TestIntegration_DryRunMovesNothing(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("x"), 0o644))

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})

	require.NoError(t, runMove(context.Background(), m, MoveOptions{DryRun: true}))

	assert.FileExists(t, filepath.Join(srcDir, "photo.jpg"))
	assert.NoFileExists(t, filepath.Join(dstDir, "photo.jpg"))
	assert.Empty(t, m.History.GetAllBatches())
}

// TestIntegration_ArchiveAction verifies action: archive packs the category into
// a single zip at the destination and keeps the originals by default.
func TestIntegration_ArchiveAction(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")
	for _, n := range []string{"a.jpg", "b.jpg"} {
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, n), []byte("data"), 0o644))
	}

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"jpg"})
	m.Categories[0].Destination.Action = models.ActionArchive
	m.Categories[0].Destination.Archive = &models.ArchiveConfig{Format: "zip", Name: "{category}"}

	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	assert.FileExists(t, filepath.Join(dstDir, "integration.zip"))
	assert.FileExists(t, filepath.Join(srcDir, "a.jpg"), "keep-source defaults to true")
}

// TestIntegration_MimeOrganizeBy verifies organize-by places a file by its real
// content type, not its (wrong) extension.
func TestIntegration_MimeOrganizeBy(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	histPath := filepath.Join(t.TempDir(), "history.json")
	// a PNG file with a misleading .jpg extension
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"),
		[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644))

	m := buildIntegrationMovelooper(t, srcDir, dstDir, histPath, []string{"all"})
	m.Categories[0].Destination.OrganizeBy = "{mime-type}/{mime-ext}"

	require.NoError(t, runMove(context.Background(), m, MoveOptions{}))

	assert.FileExists(t, filepath.Join(dstDir, "image", "png", "photo.jpg"),
		"file is placed by its real type, not its extension")
}
