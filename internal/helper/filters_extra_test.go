package helper

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MeetsAgeSizeFilters with constraints ---

func TestMeetsAgeSizeFilters_MinAgeOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MinAge: 1 * time.Hour}
	assert.True(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MinAgeFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MinAge: 1 * time.Hour}
	assert.False(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MaxAgeOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MaxAge: 1 * time.Hour}
	assert.True(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MaxAgeFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MaxAge: 1 * time.Hour}
	assert.False(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MinSizeOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	require.NoError(t, os.WriteFile(path, make([]byte, 2048), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MinSizeBytes: 1024}
	assert.True(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MinSizeFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	require.NoError(t, os.WriteFile(path, []byte("tiny"), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MinSizeBytes: 1024}
	assert.False(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MaxSizeOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	require.NoError(t, os.WriteFile(path, []byte("tiny"), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MaxSizeBytes: 1024}
	assert.True(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_MaxSizeFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	require.NoError(t, os.WriteFile(path, make([]byte, 2048), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{MaxSizeBytes: 1024}
	assert.False(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_AllConstraints_Pass(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, make([]byte, 512), 0644))

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{
		MinAge:       1 * time.Hour,
		MaxAge:       24 * time.Hour,
		MinSizeBytes: 100,
		MaxSizeBytes: 1024,
	}
	assert.True(t, MeetsAgeSizeFilters(info, f))
}

func TestMeetsAgeSizeFilters_AllConstraints_OneFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, make([]byte, 2048), 0644)) // too large

	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	f := models.CategoryFilter{
		MinAge:       1 * time.Hour,
		MaxAge:       24 * time.Hour,
		MinSizeBytes: 100,
		MaxSizeBytes: 1024, // file is 2048 — fails
	}
	assert.False(t, MeetsAgeSizeFilters(info, f))
}

// --- MatchesNameFilters edge cases ---

func TestMatchesNameFilters_RegexMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report_2024.pdf")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	import_re := mustCompileRegexHelper(t, "(?i)report")
	f := models.CategoryFilter{
		Regex:         "report",
		CompiledRegex: import_re,
	}
	assert.True(t, MatchesNameFilters("report_2024.pdf", f))
	assert.False(t, MatchesNameFilters("invoice.pdf", f))
}

func TestMatchesNameFilters_RegexNoMatch(t *testing.T) {
	re := mustCompileRegexHelper(t, "^report")
	f := models.CategoryFilter{Regex: "^report", CompiledRegex: re}
	assert.False(t, MatchesNameFilters("invoice.pdf", f))
}

func TestMatchesNameFilters_IncludePatterns(t *testing.T) {
	f := models.CategoryFilter{Include: []string{"IMG_*", "DSC_*"}}
	assert.True(t, MatchesNameFilters("IMG_001.jpg", f))
	assert.True(t, MatchesNameFilters("DSC_100.jpg", f))
	assert.False(t, MatchesNameFilters("photo.jpg", f))
}

func TestMatchesNameFilters_GlobMatch(t *testing.T) {
	f := models.CategoryFilter{Glob: "report_*.pdf"}
	assert.True(t, MatchesNameFilters("report_2024.pdf", f))
	assert.False(t, MatchesNameFilters("invoice.pdf", f))
}

// mustCompileRegexHelper compiles a regex, failing the test on error.
func mustCompileRegexHelper(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(pattern)
	require.NoError(t, err)
	return re
}
