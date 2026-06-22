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
