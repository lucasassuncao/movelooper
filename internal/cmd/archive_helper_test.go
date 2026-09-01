package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func archiveTestCategory(srcDir, dstDir string, arc *models.ArchiveConfig) *models.Category {
	enabled := true
	return &models.Category{
		Name:    "images",
		Enabled: &enabled,
		Source:  models.CategorySource{Path: srcDir, Extensions: []string{"jpg"}},
		Destination: models.CategoryDestination{
			Path:    dstDir,
			Action:  models.ActionArchive,
			Archive: arc,
		},
	}
}

func fileEntriesFrom(t *testing.T, dir string, names ...string) []scanner.FileEntry {
	t.Helper()
	for _, n := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte(n), 0o644))
	}
	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var out []scanner.FileEntry
	for _, e := range dirEntries {
		if want[e.Name()] {
			out = append(out, scanner.FileEntry{Dir: dir, Entry: e})
		}
	}
	return out
}

func TestNewArchiveProgress_NilForNonPretty(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	assert.Nil(t, newArchiveProgress(m), "no progress bar for the structured (non-tty) logger")
}

func TestArchiveCategory_WritesZipAndKeepsSourceByDefault(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	files := fileEntriesFrom(t, src, "a.jpg", "b.jpg")

	cat := archiveTestCategory(src, dst, &models.ArchiveConfig{Format: "zip", Name: "{category}"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	path, err := archiveCategory(context.Background(), m, cat, files, batch)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dst, "images.zip"), path)
	assert.FileExists(t, path)
	assert.FileExists(t, filepath.Join(src, "a.jpg"), "keep-source defaults to true")
}

func TestArchiveCategory_KeepSourceFalseDeletesOriginals(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	files := fileEntriesFrom(t, src, "a.jpg")
	del := false
	cat := archiveTestCategory(src, dst, &models.ArchiveConfig{Format: "zip", KeepSource: &del})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	path, err := archiveCategory(context.Background(), m, cat, files, batch)
	require.NoError(t, err)
	require.FileExists(t, path)
	assert.NoFileExists(t, filepath.Join(src, "a.jpg"), "keep-source:false removes originals after success")
}

// TestArchiveCategory_WriteFailureReturnsError is a regression test: a failed
// archive write must surface as an error (and ultimately a non-zero exit code),
// not just a log line.
func TestArchiveCategory_WriteFailureReturnsError(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	files := fileEntriesFrom(t, src, "a.jpg")
	// Remove the source file after scanning so archive.Write fails on open.
	require.NoError(t, os.Remove(filepath.Join(src, "a.jpg")))

	cat := archiveTestCategory(src, dst, &models.ArchiveConfig{Format: "zip"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	path, err := archiveCategory(context.Background(), m, cat, files, batch)
	assert.Error(t, err)
	assert.Empty(t, path)
}

// TestArchiveEntries_FlattenKeepsCollidingNames is a regression test: with
// flatten, two files sharing a base name in different sub-directories were
// given the same entry name. Extracting the archive kept one of them, and with
// keep-source: false both originals had already been deleted.
func TestArchiveEntries_FlattenKeepsCollidingNames(t *testing.T) {
	root := t.TempDir()
	subA := filepath.Join(root, "a")
	subB := filepath.Join(root, "b")
	require.NoError(t, os.MkdirAll(subA, 0o750))
	require.NoError(t, os.MkdirAll(subB, 0o750))
	files := append(fileEntriesFrom(t, subA, "notes.pdf"), fileEntriesFrom(t, subB, "notes.pdf")...)

	cat := archiveTestCategory(root, t.TempDir(), &models.ArchiveConfig{Format: "zip", Flatten: true})
	entries := archiveEntries(cat, files)

	require.Len(t, entries, 2)
	assert.Equal(t, []string{"notes.pdf", "notes(1).pdf"}, []string{entries[0].Name, entries[1].Name})
	assert.NotEqual(t, entries[0].Source, entries[1].Source, "both originals are still archived")
}

// TestArchiveEntries_WithoutFlattenKeepsRelativePaths confirms the de-duplication
// does not touch the normal case, where the sub-directory already separates them.
func TestArchiveEntries_WithoutFlattenKeepsRelativePaths(t *testing.T) {
	root := t.TempDir()
	subA := filepath.Join(root, "a")
	subB := filepath.Join(root, "b")
	require.NoError(t, os.MkdirAll(subA, 0o750))
	require.NoError(t, os.MkdirAll(subB, 0o750))
	files := append(fileEntriesFrom(t, subA, "notes.pdf"), fileEntriesFrom(t, subB, "notes.pdf")...)

	cat := archiveTestCategory(root, t.TempDir(), &models.ArchiveConfig{Format: "zip"})
	entries := archiveEntries(cat, files)

	assert.Equal(t, []string{"a/notes.pdf", "b/notes.pdf"}, []string{entries[0].Name, entries[1].Name})
}

// TestArchiveConflictPath_ComparisonStrategiesSaySo covers the strategies that
// cannot apply to an archive: renaming is the outcome, but silently, it looked
// like the category's own setting had been honoured.
func TestArchiveConflictPath_ComparisonStrategiesSaySo(t *testing.T) {
	dst := t.TempDir()
	existing := filepath.Join(dst, "images.zip")
	require.NoError(t, os.WriteFile(existing, []byte("old"), 0o600))

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)

	path, err := archiveConflictPath(m, models.ConflictStrategyNewest, existing)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dst, "images(1).zip"), path)
	assert.Contains(t, buf.String(), "does not apply to an archive")

	buf.Reset()
	path, err = archiveConflictPath(m, models.ConflictStrategyRename, existing)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dst, "images(1).zip"), path)
	assert.NotContains(t, buf.String(), "does not apply", "rename is what it says it is")
}
