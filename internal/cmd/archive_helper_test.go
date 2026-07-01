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

func archiveTestCategory(name, srcDir, dstDir string, arc *models.ArchiveConfig) *models.Category {
	enabled := true
	return &models.Category{
		Name:    name,
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

	cat := archiveTestCategory("images", src, dst, &models.ArchiveConfig{Format: "zip", Name: "{category}"})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	path := archiveCategory(context.Background(), m, cat, files, batch)

	assert.Equal(t, filepath.Join(dst, "images.zip"), path)
	assert.FileExists(t, path)
	assert.FileExists(t, filepath.Join(src, "a.jpg"), "keep-source defaults to true")
}

func TestArchiveCategory_KeepSourceFalseDeletesOriginals(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	files := fileEntriesFrom(t, src, "a.jpg")
	del := false
	cat := archiveTestCategory("images", src, dst, &models.ArchiveConfig{Format: "zip", KeepSource: &del})
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, []*models.Category{cat})
	batch := moveBatch{moved: make(movedSet), batchID: "batch_test", stats: &runStats{}}

	path := archiveCategory(context.Background(), m, cat, files, batch)
	require.FileExists(t, path)
	assert.NoFileExists(t, filepath.Join(src, "a.jpg"), "keep-source:false removes originals after success")
}
