package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// touch creates an empty file at path.
func touch(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))
}

func TestScan_DetectsKnownExtensions(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "photo.jpg"))
	touch(t, filepath.Join(dir, "clip.mp4"))
	touch(t, filepath.Join(dir, "song.mp3"))

	result, err := scanner.Scan(dir)
	require.NoError(t, err)

	names := make([]string, len(result.Categories))
	for i, c := range result.Categories {
		names[i] = c.Name
	}
	assert.Contains(t, names, "images")
	assert.Contains(t, names, "videos")
	assert.Contains(t, names, "audio")
}

func TestScan_CaseInsensitiveExtension(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "PHOTO.JPG"))
	touch(t, filepath.Join(dir, "doc.PDF"))

	result, err := scanner.Scan(dir)
	require.NoError(t, err)

	names := make([]string, len(result.Categories))
	for i, c := range result.Categories {
		names[i] = c.Name
	}
	assert.Contains(t, names, "images")
	assert.Contains(t, names, "documents")
}

func TestScan_EmptyDir_ReturnsNoCategories(t *testing.T) {
	dir := t.TempDir()

	result, err := scanner.Scan(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Categories)
}

func TestScan_UnknownExtensionOnly_ReturnsNoCategories(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "data.xyz123"))

	result, err := scanner.Scan(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Categories)
}

func TestScan_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0755))
	// jpg inside subdir should NOT be detected (non-recursive)
	touch(t, filepath.Join(subdir, "nested.jpg"))

	result, err := scanner.Scan(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Categories)
}

func TestScan_PreservesDictionaryOrder(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "a.mp3"))  // audio
	touch(t, filepath.Join(dir, "b.jpg"))  // images
	touch(t, filepath.Join(dir, "c.epub")) // ebooks

	result, err := scanner.Scan(dir)
	require.NoError(t, err)

	// images comes before audio, audio comes before ebooks in the dictionary.
	require.Len(t, result.Categories, 3)
	assert.Equal(t, "images", result.Categories[0].Name)
	assert.Equal(t, "audio", result.Categories[1].Name)
	assert.Equal(t, "ebooks", result.Categories[2].Name)
}

func TestScan_OnlyFoundExtensionsReturned(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "photo.jpg")) // only jpg, not all image exts

	result, err := scanner.Scan(dir)
	require.NoError(t, err)
	require.Len(t, result.Categories, 1)
	assert.Equal(t, []string{"jpg"}, result.Categories[0].Extensions)
}

func TestScan_PathDoesNotExist(t *testing.T) {
	_, err := scanner.Scan("/this/path/does/not/exist/ever")
	assert.Error(t, err)
}

func TestScan_PathIsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	touch(t, file)

	_, err := scanner.Scan(file)
	assert.Error(t, err)
}
