package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
