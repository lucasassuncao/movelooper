package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkdirAll(t *testing.T, base, rel string) string {
	t.Helper()
	p := filepath.Join(base, rel)
	require.NoError(t, os.MkdirAll(p, 0755))
	return p
}

func src(path string, opts ...func(*models.CategorySource)) models.CategorySource {
	s := models.CategorySource{Path: path}
	for _, o := range opts {
		o(&s)
	}
	return s
}

func withRecursive(s *models.CategorySource) { s.Recursive = true }

func withMaxDepth(n int) func(*models.CategorySource) {
	return func(s *models.CategorySource) { s.MaxDepth = n }
}

func withExclude(paths ...string) func(*models.CategorySource) {
	return func(s *models.CategorySource) { s.ExcludePaths = paths }
}

func entryNames(entries []scanner.FileEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Entry.Name()
	}
	return names
}

func TestWalkSource_NonRecursive_OnlyTopLevel(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.pdf"))
	sub := mkdirAll(t, root, "sub")
	touch(t, filepath.Join(sub, "b.pdf"))

	entries, err := scanner.WalkSource(src(root), nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "a.pdf", entries[0].Entry.Name())
	assert.Equal(t, root, entries[0].Dir)
}

func TestWalkSource_Recursive_IncludesSubdirs(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.pdf"))
	sub := mkdirAll(t, root, "sub")
	touch(t, filepath.Join(sub, "b.pdf"))

	entries, err := scanner.WalkSource(src(root, withRecursive), nil)
	require.NoError(t, err)

	names := entryNames(entries)
	assert.Contains(t, names, "a.pdf")
	assert.Contains(t, names, "b.pdf")
}

func TestWalkSource_MaxDepth_LimitsDescend(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "depth0.txt"))
	sub1 := mkdirAll(t, root, "sub1")
	touch(t, filepath.Join(sub1, "depth1.txt"))
	sub2 := mkdirAll(t, root, "sub1/sub2")
	touch(t, filepath.Join(sub2, "depth2.txt"))

	entries, err := scanner.WalkSource(src(root, withRecursive, withMaxDepth(1)), nil)
	require.NoError(t, err)

	names := entryNames(entries)
	assert.Contains(t, names, "depth0.txt")
	assert.Contains(t, names, "depth1.txt")
	assert.NotContains(t, names, "depth2.txt")
}

func TestWalkSource_AutoExclude_SkipsDestination(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.pdf"))
	dest := mkdirAll(t, root, "Sorted")
	touch(t, filepath.Join(dest, "already-moved.pdf"))

	entries, err := scanner.WalkSource(src(root, withRecursive), []string{dest})
	require.NoError(t, err)

	names := entryNames(entries)
	assert.Contains(t, names, "a.pdf")
	assert.NotContains(t, names, "already-moved.pdf")
}

func TestWalkSource_ExcludePaths_SkipsUserDefinedDirs(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.pdf"))
	archive := mkdirAll(t, root, "Archive")
	touch(t, filepath.Join(archive, "old.pdf"))

	entries, err := scanner.WalkSource(src(root, withRecursive, withExclude(archive)), nil)
	require.NoError(t, err)

	names := entryNames(entries)
	assert.Contains(t, names, "a.pdf")
	assert.NotContains(t, names, "old.pdf")
}

func TestWalkSource_Recursive_SkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real.pdf")
	touch(t, real)
	link := filepath.Join(root, "link.pdf")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink creation not supported:", err)
	}

	entries, err := scanner.WalkSource(src(root, withRecursive), nil)
	require.NoError(t, err)

	names := entryNames(entries)
	assert.Contains(t, names, "real.pdf")
	assert.NotContains(t, names, "link.pdf")
}

func TestWalkSource_EmptyDir_ReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	entries, err := scanner.WalkSource(src(root, withRecursive), nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestWalkSource_InvalidPath_ReturnsError(t *testing.T) {
	_, err := scanner.WalkSource(src("/nonexistent/path/abc123"), nil)
	assert.Error(t, err)
}

func TestWalkSource_FileEntryDir_IsAbsolute(t *testing.T) {
	root := t.TempDir()
	sub := mkdirAll(t, root, "sub")
	touch(t, filepath.Join(sub, "file.txt"))

	entries, err := scanner.WalkSource(src(root, withRecursive), nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, filepath.IsAbs(entries[0].Dir))
	assert.Equal(t, sub, entries[0].Dir)
}
