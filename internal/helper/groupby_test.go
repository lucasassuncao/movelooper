package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSizeRange(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "tiny"},
		{500 * 1024, "tiny"},           // 500 KB
		{sizeThresholdTiny, "small"},   // 1 MB
		{50 * 1024 * 1024, "small"},    // 50 MB
		{sizeThresholdSmall, "medium"}, // 100 MB
		{500 * 1024 * 1024, "medium"},  // 500 MB
		{sizeThresholdMedium, "large"}, // 1 GB
		{2 * 1024 * 1024 * 1024, "large"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, fileSizeRange(tt.size), "size=%d", tt.size)
	}
}

func TestResolveGroupBy_EmptyTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("", info, "docs", time.Now())
	assert.Empty(t, result)
}

func TestResolveGroupBy_ExtToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "document.PDF")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("{ext}", info, "docs", time.Now())
	assert.Equal(t, "pdf", result)
}

func TestResolveGroupBy_ExtUpperToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.jpg")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("{ext-upper}", info, "photos", time.Now())
	assert.Equal(t, "JPG", result)
}

func TestResolveGroupBy_CategoryToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("{category}", info, "MyCategory", time.Now())
	assert.Equal(t, "MyCategory", result)
}

func TestResolveGroupBy_RunDateTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	now := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	result := ResolveGroupBy("{year}/{month}/{day}", info, "cat", now)
	assert.Equal(t, filepath.FromSlash("2024/03/15"), result)
}

func TestResolveGroupBy_ModDateTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	modTime := time.Date(2023, 7, 4, 12, 0, 0, 0, time.Local)
	require.NoError(t, os.Chtimes(path, modTime, modTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("{mod-year}/{mod-month}/{mod-day}", info, "cat", time.Now())
	assert.Equal(t, filepath.FromSlash("2023/07/04"), result)
}

func TestResolveGroupBy_SizeRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 500), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	result := ResolveGroupBy("{size-range}", info, "bin", time.Now())
	assert.Equal(t, "tiny", result)
}

func TestResolveGroupBy_CombinedTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	now := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	result := ResolveGroupBy("{category}/{year}/{ext}", info, "docs", now)
	assert.Equal(t, filepath.FromSlash("docs/2024/pdf"), result)
}
