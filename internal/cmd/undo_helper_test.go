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

	err := undoBatch(context.Background(), m, "batch_missing", false, false, nil)
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

	require.NoError(t, undoBatch(context.Background(), m, "batch_dry", true, false, nil))

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
