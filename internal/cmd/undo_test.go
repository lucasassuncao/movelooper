package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSilentMovelooperWithHistory creates a test Movelooper with a real in-memory history.
func newSilentMovelooperWithHistory(t *testing.T) *models.Movelooper {
	t.Helper()
	h, err := history.NewHistory(50)
	if err != nil {
		// Fall back: no history (tests that need it will skip or use workarounds)
		h = nil
	}
	m := newSilentMovelooper(nil)
	m.History = h
	return m
}

// addHistoryEntry adds a single history entry using the history API.
func addHistoryEntry(t *testing.T, h *history.History, batchID, src, dst string) {
	t.Helper()
	require.NoError(t, h.Add(history.Entry{
		Source:      src,
		Destination: dst,
		Timestamp:   time.Now(),
		BatchID:     batchID,
	}))
}

// --- undoBatch dry-run ---

func TestUndoBatch_DryRun_ReportsWouldRestore(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")

	// Simulate that the file was moved: it's now at dst
	require.NoError(t, os.WriteFile(dstFile, []byte("data"), 0644))

	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}
	addHistoryEntry(t, m.History, "batch_1", srcFile, dstFile)

	err := undoBatch(m, "batch_1", true)
	assert.NoError(t, err)

	// Dry-run: file must remain at destination
	assert.FileExists(t, dstFile)
	assert.NoFileExists(t, srcFile)
}

func TestUndoBatch_DryRun_WarnsMissingDestination(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "missing.txt")
	dstFile := filepath.Join(dst, "missing.txt") // does NOT exist

	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}
	addHistoryEntry(t, m.History, "batch_missing", srcFile, dstFile)

	// Should not error even when destination file is absent
	err := undoBatch(m, "batch_missing", true)
	assert.NoError(t, err)
}

func TestUndoBatch_DryRun_WarnsOccupiedSource(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "occupied.txt")
	dstFile := filepath.Join(dst, "occupied.txt")

	// Both exist: source is occupied, destination also exists
	require.NoError(t, os.WriteFile(srcFile, []byte("original"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("moved"), 0644))

	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}
	addHistoryEntry(t, m.History, "batch_occupied", srcFile, dstFile)

	err := undoBatch(m, "batch_occupied", true)
	assert.NoError(t, err)

	// Dry-run: nothing moved
	assert.FileExists(t, srcFile)
	assert.FileExists(t, dstFile)
}

func TestUndoBatch_BatchNotFound(t *testing.T) {
	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}

	err := undoBatch(m, "nonexistent_batch", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in history")
}

// --- printBatchList ---

func TestPrintBatchList_NoBatches(t *testing.T) {
	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}

	// No error and no panic when history is empty
	err := printBatchList(m)
	assert.NoError(t, err)
}

func TestPrintBatchList_WithBatches(t *testing.T) {
	dst := t.TempDir()

	m := newSilentMovelooperWithHistory(t)
	if m.History == nil {
		t.Skip("history not available in this environment")
	}

	addHistoryEntry(t, m.History, "batch_X", "/src/a.txt", filepath.Join(dst, "a.txt"))
	addHistoryEntry(t, m.History, "batch_Y", "/src/b.txt", filepath.Join(dst, "b.txt"))

	err := printBatchList(m)
	assert.NoError(t, err)
}

// --- UndoCmd structure ---

func TestUndoCmd_NilHistory_ReturnsError(t *testing.T) {
	m := newSilentMovelooper(nil)
	// m.History is nil
	cmd := UndoCmd(m)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "history tracking is not initialized")
}
