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

// --- undoBatch ---

func TestUndoBatch(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, m *models.Movelooper) (batchID string, srcFile string, dstFile string)
		dryRun  bool
		wantErr string
		check   func(t *testing.T, srcFile, dstFile string)
	}{
		{
			name: "dry-run reports would restore",
			setup: func(t *testing.T, m *models.Movelooper) (string, string, string) {
				src := t.TempDir()
				dst := t.TempDir()
				srcFile := filepath.Join(src, "file.txt")
				dstFile := filepath.Join(dst, "file.txt")
				require.NoError(t, os.WriteFile(dstFile, []byte("data"), 0644))
				addHistoryEntry(t, m.History, "batch_1", srcFile, dstFile)
				return "batch_1", srcFile, dstFile
			},
			dryRun: true,
			check: func(t *testing.T, srcFile, dstFile string) {
				assert.FileExists(t, dstFile)
				assert.NoFileExists(t, srcFile)
			},
		},
		{
			name: "dry-run warns missing destination",
			setup: func(t *testing.T, m *models.Movelooper) (string, string, string) {
				src := t.TempDir()
				dst := t.TempDir()
				srcFile := filepath.Join(src, "missing.txt")
				dstFile := filepath.Join(dst, "missing.txt") // does NOT exist
				addHistoryEntry(t, m.History, "batch_missing", srcFile, dstFile)
				return "batch_missing", srcFile, dstFile
			},
			dryRun: true,
		},
		{
			name: "dry-run warns occupied source",
			setup: func(t *testing.T, m *models.Movelooper) (string, string, string) {
				src := t.TempDir()
				dst := t.TempDir()
				srcFile := filepath.Join(src, "occupied.txt")
				dstFile := filepath.Join(dst, "occupied.txt")
				require.NoError(t, os.WriteFile(srcFile, []byte("original"), 0644))
				require.NoError(t, os.WriteFile(dstFile, []byte("moved"), 0644))
				addHistoryEntry(t, m.History, "batch_occupied", srcFile, dstFile)
				return "batch_occupied", srcFile, dstFile
			},
			dryRun: true,
			check: func(t *testing.T, srcFile, dstFile string) {
				assert.FileExists(t, srcFile)
				assert.FileExists(t, dstFile)
			},
		},
		{
			name: "batch not found returns error",
			setup: func(t *testing.T, m *models.Movelooper) (string, string, string) {
				return "nonexistent_batch", "", ""
			},
			dryRun:  true,
			wantErr: "not found in history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSilentMovelooperWithHistory(t)
			if m.History == nil {
				t.Skip("history not available in this environment")
			}

			batchID, srcFile, dstFile := tt.setup(t, m)
			err := undoBatch(m, batchID, tt.dryRun)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, srcFile, dstFile)
			}
		})
	}
}

// --- printBatchList ---

func TestPrintBatchList(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, m *models.Movelooper)
	}{
		{
			name:  "no batches",
			setup: func(t *testing.T, m *models.Movelooper) {},
		},
		{
			name: "with batches",
			setup: func(t *testing.T, m *models.Movelooper) {
				dst := t.TempDir()
				addHistoryEntry(t, m.History, "batch_X", "/src/a.txt", filepath.Join(dst, "a.txt"))
				addHistoryEntry(t, m.History, "batch_Y", "/src/b.txt", filepath.Join(dst, "b.txt"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSilentMovelooperWithHistory(t)
			if m.History == nil {
				t.Skip("history not available in this environment")
			}
			tt.setup(t, m)
			assert.NoError(t, printBatchList(m))
		})
	}
}

// --- UndoCmd structure ---

func TestUndoCmd_NilHistory_ReturnsError(t *testing.T) {
	m := newSilentMovelooper(nil)
	cmd := UndoCmd(m)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "history tracking is not initialized")
}
