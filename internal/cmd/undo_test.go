package cmd

import (
	"bytes"
	"io"
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

// addHistoryEntryWithCategory adds a history entry that includes the category name.
func addHistoryEntryWithCategory(t *testing.T, h *history.History, batchID, src, dst, category string) {
	t.Helper()
	require.NoError(t, h.Add(history.Entry{
		Source:      src,
		Destination: dst,
		Timestamp:   time.Now(),
		BatchID:     batchID,
		Action:      "move",
		Category:    category,
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
			check: func(t *testing.T, srcFile, dstFile string) {
				// Neither file should exist - nothing was created or moved
				assert.NoFileExists(t, srcFile)
				assert.NoFileExists(t, dstFile)
			},
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
			err := undoBatch(m, batchID, tt.dryRun, nil)

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
		name      string
		setup     func(t *testing.T, m *models.Movelooper)
		wantInOut []string // substrings expected in stdout
	}{
		{
			name:  "no batches",
			setup: func(t *testing.T, m *models.Movelooper) {},
			// empty history: nothing printed to stdout (logger.Info used instead)
		},
		{
			name: "with batches prints batch IDs",
			setup: func(t *testing.T, m *models.Movelooper) {
				dst := t.TempDir()
				addHistoryEntry(t, m.History, "batch_X", "/src/a.txt", filepath.Join(dst, "a.txt"))
				addHistoryEntry(t, m.History, "batch_Y", "/src/b.txt", filepath.Join(dst, "b.txt"))
			},
			wantInOut: []string{"batch_X", "batch_Y", "BATCH ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSilentMovelooperWithHistory(t)
			if m.History == nil {
				t.Skip("history not available in this environment")
			}

			// Redirect stdout to capture tabwriter output.
			r, w, err := os.Pipe()
			require.NoError(t, err)
			origStdout := os.Stdout
			os.Stdout = w
			t.Cleanup(func() { os.Stdout = origStdout })

			tt.setup(t, m)
			require.NoError(t, printBatchList(m))

			w.Close()
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			out := buf.String()

			for _, want := range tt.wantInOut {
				assert.Contains(t, out, want)
			}
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

func TestUndoCopyOrSymlink_RemovesDst(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.txt")
	dst := filepath.Join(dir, "copy.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(dst, []byte("hello"), 0644))

	require.NoError(t, undoCopyOrSymlink(dst))
	assert.NoFileExists(t, dst)
	assert.FileExists(t, src)
}

func TestUndoSymlink_RemovesLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.txt")
	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))
	if err := os.Symlink(src, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	require.NoError(t, undoCopyOrSymlink(link))
	_, err := os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "link should be removed")
	assert.FileExists(t, src)
}

func TestUndoBatch_CopyDryRun(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "copy.txt")
	require.NoError(t, os.WriteFile(dst, []byte("hello"), 0644))

	m := newSilentMovelooperWithHistory(t)
	require.NoError(t, m.History.Add(history.Entry{
		Source:      filepath.Join(dir, "original.txt"),
		Destination: dst,
		BatchID:     "batch_copy",
		Action:      "copy",
		Timestamp:   time.Now(),
	}))

	require.NoError(t, undoBatch(m, "batch_copy", true, nil))
	assert.FileExists(t, dst) // dry-run must not remove the file
}

// --- undoBatch partial by category ---

func TestUndoBatch_PartialByCategory(t *testing.T) {
	t.Run("dry-run reports only matching category files", func(t *testing.T) {
		m := newSilentMovelooperWithHistory(t)

		dst1 := t.TempDir()
		imgDst := filepath.Join(dst1, "photo.jpg")
		require.NoError(t, os.WriteFile(imgDst, []byte("img"), 0644))

		dst2 := t.TempDir()
		docDst := filepath.Join(dst2, "report.pdf")
		require.NoError(t, os.WriteFile(docDst, []byte("doc"), 0644))

		addHistoryEntryWithCategory(t, m.History, "partial_dry_b1", "/src/photo.jpg", imgDst, "images")
		addHistoryEntryWithCategory(t, m.History, "partial_dry_b1", "/src/report.pdf", docDst, "docs")

		err := undoBatch(m, "partial_dry_b1", true, []string{"images"})
		require.NoError(t, err)

		// dry-run: no files moved
		assert.FileExists(t, imgDst)
		assert.FileExists(t, docDst)
	})

	t.Run("no entries for category warns and returns without error", func(t *testing.T) {
		m := newSilentMovelooperWithHistory(t)

		dst := t.TempDir()
		docDst := filepath.Join(dst, "report.pdf")
		require.NoError(t, os.WriteFile(docDst, []byte("doc"), 0644))

		addHistoryEntryWithCategory(t, m.History, "partial_nomatch_b1", "/src/report.pdf", docDst, "docs")

		err := undoBatch(m, "partial_nomatch_b1", true, []string{"images"})
		require.NoError(t, err)

		// batch untouched
		assert.FileExists(t, docDst)
		assert.Len(t, m.History.GetBatch("partial_nomatch_b1"), 1)
	})

	t.Run("entry with empty Category is skipped when category filter is active", func(t *testing.T) {
		m := newSilentMovelooperWithHistory(t)

		dst := t.TempDir()
		fileDst := filepath.Join(dst, "file.txt")
		require.NoError(t, os.WriteFile(fileDst, []byte("data"), 0644))

		// legacy entry without Category
		addHistoryEntry(t, m.History, "partial_legacy_b1", "/src/file.txt", fileDst)

		err := undoBatch(m, "partial_legacy_b1", true, []string{"images"})
		require.NoError(t, err)

		// no match found - file untouched
		assert.FileExists(t, fileDst)
	})

	t.Run("unknown batch with category filter returns error", func(t *testing.T) {
		m := newSilentMovelooperWithHistory(t)

		addHistoryEntryWithCategory(t, m.History, "partial_known_b1", "/src/a.txt", "/dst/a.txt", "images")

		err := undoBatch(m, "partial_unknown_b99", true, []string{"images"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in history")
	})
}
