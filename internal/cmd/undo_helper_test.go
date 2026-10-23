package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreEntries_SkipsArchiveBatch(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      "/src",
		Destination: "/dst/images.zip",
		Action:      string(models.ActionArchive),
		BatchID:     "batch_x",
		Category:    "images",
	}}
	restored := restoreEntries(context.Background(), m, entries, false)
	assert.Empty(t, restored, "archive entries are not restored")
	assert.Contains(t, buf.String(), "archive")
}

// TestRestoreEntries_CopyRemovesDestination is a regression test: undo of a
// copy/symlink batch must remove the destination even though the source still
// exists (copy never consumes the source), instead of skipping with
// "source location already occupied".
func TestRestoreEntries_CopyRemovesDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(dst, []byte("x"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      src,
		Destination: dst,
		Action:      string(models.ActionCopy),
		BatchID:     "batch_x",
		Category:    "docs",
		Timestamp:   time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, false)
	assert.Len(t, restored, 1, "copy entry must be restored")
	_, err := os.Stat(dst)
	assert.True(t, os.IsNotExist(err), "destination must be removed")
	_, err = os.Stat(src)
	assert.NoError(t, err, "source must be untouched")
}

// TestRestoreEntries_MoveSkipsWhenSourceOccupied keeps the original guard for
// move undo: if something new occupies the source path, do not overwrite it.
func TestRestoreEntries_MoveSkipsWhenSourceOccupied(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(src, []byte("new file"), 0o600))
	require.NoError(t, os.WriteFile(dst, []byte("moved"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      src,
		Destination: dst,
		Action:      string(models.ActionMove),
		BatchID:     "batch_x",
		Category:    "docs",
	}}

	restored := restoreEntries(context.Background(), m, entries, false)
	assert.Empty(t, restored)
	assert.Contains(t, buf.String(), "source location already occupied")
}

// TestRestoreEntries_CopyKeepsLastCopy is a regression test: undo of a copy
// deletes the destination, so when the original is no longer at the source that
// deletion would destroy the only remaining copy of the file.
func TestRestoreEntries_CopyKeepsLastCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("only copy"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      src, // never created: the original is gone
		Destination: dst,
		Action:      string(models.ActionCopy),
		BatchID:     "batch_x",
		Category:    "docs",
		Timestamp:   time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, false)
	assert.Empty(t, restored)
	assert.FileExists(t, dst, "the last copy must survive the undo")
	assert.Contains(t, buf.String(), "last copy")
}

// TestRestoreEntries_CopyForceRemovesLastCopy verifies --force overrides the
// guard, since the config is the user's call once they have been told.
func TestRestoreEntries_CopyForceRemovesLastCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("only copy"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      src,
		Destination: dst,
		Action:      string(models.ActionCopy),
		BatchID:     "batch_x",
		Category:    "docs",
		Timestamp:   time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, true)
	assert.Len(t, restored, 1)
	assert.NoFileExists(t, dst)
}

// TestRestoreEntries_CopyKeepsModifiedDestination covers the second way undo of
// a copy destroys data: the copy was edited after the run, so removing it
// discards work that exists nowhere else.
func TestRestoreEntries_CopyKeepsModifiedDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(dst, []byte("edited since"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      src,
		Destination: dst,
		Action:      string(models.ActionCopy),
		BatchID:     "batch_x",
		Category:    "docs",
		Timestamp:   time.Now().Add(-time.Hour), // the copy is newer than its own run
	}}

	restored := restoreEntries(context.Background(), m, entries, false)
	assert.Empty(t, restored)
	assert.FileExists(t, dst)
	assert.Contains(t, buf.String(), "modified after it was copied")
}

func TestUndoBatch_UnknownBatchFails(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)

	err := undoBatch(context.Background(), m, "batch_missing", undoOptions{})
	assert.ErrorContains(t, err, "not found in history")
}

// TestUndoBatch_DryRunTouchesNothing verifies the preview leaves both the files
// and the history exactly as they were.
func TestUndoBatch_DryRunTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("moved"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entry := history.Entry{
		Source:      src,
		Destination: dst,
		Action:      string(models.ActionMove),
		BatchID:     "batch_dry",
		Category:    "docs",
		Timestamp:   time.Now(),
	}
	require.NoError(t, m.History.AddBatch([]history.Entry{entry}))

	require.NoError(t, undoBatch(context.Background(), m, "batch_dry", undoOptions{DryRun: true}))

	assert.FileExists(t, dst, "dry-run must not move the file back")
	assert.NoFileExists(t, src)
	assert.Len(t, m.History.GetBatch("batch_dry"), 1, "dry-run must not consume the batch")
	assert.Contains(t, buf.String(), "[dry-run] would restore file(s)")
}

// TestDryRunUndoBatch_ReportsCopyGuard keeps the preview honest: it must report
// the copies undo will refuse to delete, not list them as removals.
func TestDryRunUndoBatch_ReportsCopyGuard(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("only copy"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      filepath.Join(dir, "src", "file.txt"), // gone
		Destination: dst,
		Action:      string(models.ActionCopy),
		BatchID:     "batch_x",
		Category:    "docs",
		Timestamp:   time.Now(),
	}}

	require.NoError(t, dryRunUndoBatch(m, "batch_x", entries, false))
	out := buf.String()
	assert.Contains(t, out, "would keep the copy")
	assert.NotContains(t, out, "would remove file(s)")
}

func TestFilterEntriesByCategory(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	all := []history.Entry{
		{Source: "/a", Category: "images", BatchID: "b"},
		{Source: "/b", Category: "docs", BatchID: "b"},
		{Source: "/c", Category: "", BatchID: "b"}, // recorded before category tracking
	}

	filtered, ok := filterEntriesByCategory(m, "b", all, []string{"images"})
	require.True(t, ok)
	require.Len(t, filtered, 1)
	assert.Equal(t, "/a", filtered[0].Source)
	assert.Contains(t, buf.String(), "unknown category")

	filtered, ok = filterEntriesByCategory(m, "b", all, []string{"videos"})
	assert.False(t, ok, "no matching category means there is nothing to undo")
	assert.Nil(t, filtered)
	assert.Contains(t, buf.String(), "no entries for the specified categories")
}

// noTerminal makes stdin read as not a terminal, as under cron or a pipe.
func noTerminal(t *testing.T) {
	t.Helper()
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = old })
}

// movedEntry puts a file at its destination and records the move in batchID.
func movedEntry(t *testing.T, m *models.Movelooper, batchID string) (src, dst string) {
	t.Helper()
	dir := t.TempDir()
	src = filepath.Join(dir, "src", "file.txt")
	dst = filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("moved"), 0o600))
	require.NoError(t, m.History.AddBatch([]history.Entry{{
		Source: src, Destination: dst, Action: string(models.ActionMove),
		BatchID: batchID, Category: "docs", Timestamp: time.Now(),
	}}))
	return src, dst
}

// Without a terminal nobody can answer the confirmation: undo must say so and
// stop, not sit in a prompt that spins on a closed input.
func TestUndoBatch_WithoutATerminalAsksForYes(t *testing.T) {
	noTerminal(t)
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	_, dst := movedEntry(t, m, "batch_tty")

	done := make(chan error, 1)
	go func() { done <- undoBatch(context.Background(), m, "batch_tty", undoOptions{}) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "--yes")
	case <-time.After(5 * time.Second):
		t.Fatal("undo waited on a prompt nobody can answer")
	}
	assert.FileExists(t, dst, "nothing is restored without a confirmation")
}

func TestUndoBatch_YesRestoresWithoutAsking(t *testing.T) {
	noTerminal(t)
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	src, dst := movedEntry(t, m, "batch_yes")

	require.NoError(t, undoBatch(context.Background(), m, "batch_yes", undoOptions{Yes: true}))

	assert.FileExists(t, src)
	assert.NoFileExists(t, dst)
}

// Scripts read the exit code: an undo that restored nothing used to exit 0.
func TestUndoBatch_ReportsFilesItCouldNotRestore(t *testing.T) {
	noTerminal(t)
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	_, dst := movedEntry(t, m, "batch_gone")
	require.NoError(t, os.Remove(dst))

	err := undoBatch(context.Background(), m, "batch_gone", undoOptions{Yes: true})

	require.ErrorContains(t, err, "1 file(s) could not be restored")
	assert.Len(t, m.History.GetBatch("batch_gone"), 1, "a failed restore stays in history for a retry")
}

// Undoing a symlink removes the link, never a real file that has since taken
// its name: that one movelooper did not make.
func TestRestoreEntries_SymlinkKeepsAFileThatReplacedTheLink(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("a real file now"), 0o600))
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source: filepath.Join(dir, "src", "file.txt"), Destination: dst,
		Action: string(models.ActionSymlink), BatchID: "batch_l", Category: "docs", Timestamp: time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, false)

	assert.Empty(t, restored)
	assert.FileExists(t, dst)
}

// A link whose original is gone still holds the name: Stat followed it, said
// "not found" and left the dangling link behind for good.
func TestRestoreEntries_SymlinkRemovesADanglingLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	if err := os.Symlink(src, dst); err != nil {
		t.Skipf("symlinks not available here: %v", err)
	}
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source: src, Destination: dst,
		Action: string(models.ActionSymlink), BatchID: "batch_d", Category: "docs", Timestamp: time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, false)

	assert.Len(t, restored, 1)
	_, err := os.Lstat(dst)
	assert.True(t, os.IsNotExist(err), "the dangling link is gone")
}

// A duplicate hash_check removed comes back as a copy of the identical file
// at the destination, which stays where it is.
func TestRestoreEntries_DedupeCopiesTheDuplicateBack(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file.txt")
	dst := filepath.Join(dir, "dst", "file.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o750))
	require.NoError(t, os.WriteFile(dst, []byte("same"), 0o600))
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source: src, Destination: dst, Action: history.ActionDedupe,
		BatchID: "batch_h", Category: "docs", Timestamp: time.Now(),
	}}

	restored := restoreEntries(context.Background(), m, entries, false)

	assert.Len(t, restored, 1)
	got, err := os.ReadFile(src)
	require.NoError(t, err)
	assert.Equal(t, "same", string(got))
	assert.FileExists(t, dst, "the destination keeps its file")
}

// Undo can return the file a run moved, not the one it replaced: say so.
func TestRestoreEntries_WarnsAboutAReplacedFile(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	src, dst := movedEntry(t, m, "batch_r")
	entries := m.History.GetBatch("batch_r")
	entries[0].Replaced = true

	restored := restoreEntries(context.Background(), m, entries, false)

	assert.Len(t, restored, 1)
	assert.FileExists(t, src)
	assert.NoFileExists(t, dst)
	assert.Contains(t, buf.String(), "replaced")
}
