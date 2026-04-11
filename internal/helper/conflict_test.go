package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, content, 0644))
}

// --- getUniqueDestinationPath ---

func TestGetUniqueDestinationPath_NoConflict(t *testing.T) {
	dir := t.TempDir()
	got, err := getUniqueDestinationPath(dir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "file.txt"), got)
}

func TestGetUniqueDestinationPath_Conflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "file.txt"), []byte("original"))

	got, err := getUniqueDestinationPath(dir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "file(1).txt"), got)
}

func TestGetUniqueDestinationPath_MultipleConflicts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "file.txt"), []byte("1"))
	writeFile(t, filepath.Join(dir, "file(1).txt"), []byte("2"))
	writeFile(t, filepath.Join(dir, "file(2).txt"), []byte("3"))

	got, err := getUniqueDestinationPath(dir, "file.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "file(3).txt"), got)
}

// --- resolveConflict dispatch ---

func TestResolveConflict_UnknownFallsToRename(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("src"))
	writeFile(t, dst, []byte("dst"))

	path, shouldMove, err := resolveConflict("unknown_strategy", src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, filepath.Join(dir, "dst(1).txt"), path)
}

// --- renameResolver ---

func TestRenameResolver(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "file.txt"), []byte("existing"))

	r := &renameResolver{}
	path, shouldMove, err := r.Resolve("", "", dir, "file.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, filepath.Join(dir, "file(1).txt"), path)
}

// --- overwriteResolver ---

func TestOverwriteResolver_RemovesDst(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "file.txt")
	writeFile(t, dst, []byte("old"))

	r := &overwriteResolver{}
	path, shouldMove, err := r.Resolve("", dst, "", "")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, dst, path)
	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr))
}

// --- skipResolver ---

func TestSkipResolver(t *testing.T) {
	r := &skipResolver{}
	path, shouldMove, err := r.Resolve("", "dst", "", "")
	require.NoError(t, err)
	assert.False(t, shouldMove)
	assert.Empty(t, path)
}

// --- hashCheckResolver ---

func TestHashCheckResolver_DuplicateRemovesSrc(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("identical content")
	writeFile(t, src, content)
	writeFile(t, dst, content)

	r := &hashCheckResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.False(t, shouldMove)
	assert.Empty(t, path)
	// src must be removed (dedup)
	_, statErr := os.Stat(src)
	assert.True(t, os.IsNotExist(statErr))
}

func TestHashCheckResolver_DifferentRenames(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("content A"))
	writeFile(t, dst, []byte("content B"))

	r := &hashCheckResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, filepath.Join(dir, "dst(1).txt"), path)
}

// --- newestResolver ---

func TestNewestResolver_SrcNewer(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("src"))
	writeFile(t, dst, []byte("dst"))

	// Make dst older
	old := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(dst, old, old))

	r := &newestResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, dst, path)
}

func TestNewestResolver_DstNewer(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("src"))
	writeFile(t, dst, []byte("dst"))

	// Make src older
	old := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(src, old, old))

	r := &newestResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.False(t, shouldMove)
	assert.Empty(t, path)
}

// --- oldestResolver ---

func TestOldestResolver_SrcOlder(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("src"))
	writeFile(t, dst, []byte("dst"))

	old := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(src, old, old))

	r := &oldestResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, dst, path)
}

func TestOldestResolver_DstOlder(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, []byte("src"))
	writeFile(t, dst, []byte("dst"))

	old := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(dst, old, old))

	r := &oldestResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.False(t, shouldMove)
	assert.Empty(t, path)
}

// --- largerResolver ---

func TestLargerResolver_SrcLarger(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, make([]byte, 200))
	writeFile(t, dst, make([]byte, 100))

	r := &largerResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, dst, path)
}

func TestLargerResolver_DstLarger(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, make([]byte, 100))
	writeFile(t, dst, make([]byte, 200))

	r := &largerResolver{}
	_, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.False(t, shouldMove)
}

// --- smallerResolver ---

func TestSmallerResolver_SrcSmaller(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, make([]byte, 100))
	writeFile(t, dst, make([]byte, 200))

	r := &smallerResolver{}
	path, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.True(t, shouldMove)
	assert.Equal(t, dst, path)
}

func TestSmallerResolver_DstSmaller(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, make([]byte, 200))
	writeFile(t, dst, make([]byte, 100))

	r := &smallerResolver{}
	_, shouldMove, err := r.Resolve(src, dst, dir, "dst.txt")
	require.NoError(t, err)
	assert.False(t, shouldMove)
}
