package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

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
	restored := restoreEntries(context.Background(), m, entries)
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
	}}

	restored := restoreEntries(context.Background(), m, entries)
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

	restored := restoreEntries(context.Background(), m, entries)
	assert.Empty(t, restored)
	assert.Contains(t, buf.String(), "source location already occupied")
}
