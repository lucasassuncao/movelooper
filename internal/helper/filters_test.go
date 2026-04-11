package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MatchesIgnorePatterns ---

func TestMatchesIgnorePatterns(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		patterns      []string
		caseSensitive bool
		want          bool
	}{
		{"no patterns", "file.txt", nil, false, false},
		{"simple match", "file.txt", []string{"*.txt"}, false, true},
		{"no match", "file.go", []string{"*.txt"}, false, false},
		{"case insensitive match", "FILE.TXT", []string{"*.txt"}, false, true},
		{"case sensitive no match", "FILE.TXT", []string{"*.txt"}, true, false},
		{"case sensitive match", "file.txt", []string{"*.txt"}, true, true},
		{"multiple patterns first matches", "file.txt", []string{"*.txt", "*.go"}, false, true},
		{"multiple patterns second matches", "file.go", []string{"*.txt", "*.go"}, false, true},
		{"invalid pattern skipped", "file.txt", []string{"["}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesIgnorePatterns(tt.fileName, tt.patterns, tt.caseSensitive)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- expandGlobPattern ---

func TestExpandGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    []string
	}{
		{"*.txt", []string{"*.txt"}},
		{"*.{jpg,png}", []string{"*.jpg", "*.png"}},
		{"file.{go,py,js}", []string{"file.go", "file.py", "file.js"}},
		{"{a,b}", []string{"a", "b"}},
		{"no braces", []string{"no braces"}},
		// spaces around alternatives are trimmed
		{"*.{ jpg , png }", []string{"*.jpg", "*.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := expandGlobPattern(tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- MatchesGlob ---

func TestMatchesGlob(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		pattern       string
		caseSensitive bool
		want          bool
	}{
		{"simple match", "photo.jpg", "*.jpg", false, true},
		{"brace expansion match", "photo.jpg", "*.{jpg,png}", false, true},
		{"brace expansion second", "photo.png", "*.{jpg,png}", false, true},
		{"no match", "photo.gif", "*.{jpg,png}", false, false},
		{"case insensitive", "PHOTO.JPG", "*.jpg", false, true},
		{"case sensitive no match", "PHOTO.JPG", "*.jpg", true, false},
		{"wildcard name", "report_2024.pdf", "report_*.pdf", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesGlob(tt.fileName, tt.pattern, tt.caseSensitive)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ValidateGlob ---

func TestValidateGlob(t *testing.T) {
	assert.NoError(t, ValidateGlob("*.txt"))
	assert.NoError(t, ValidateGlob("*.{jpg,png}"))
	assert.Error(t, ValidateGlob("[invalid"))
}

// --- HasExtension ---

func TestHasExtension(t *testing.T) {
	dir := t.TempDir()
	createTempFile(t, dir, "doc.pdf")
	createTempFile(t, dir, "image.PNG")
	createTempFile(t, dir, "noext")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	byName := make(map[string]os.DirEntry)
	for _, e := range entries {
		byName[e.Name()] = e
	}

	assert.True(t, HasExtension(byName["doc.pdf"], "pdf"))
	assert.True(t, HasExtension(byName["image.PNG"], "png"))
	assert.True(t, HasExtension(byName["doc.pdf"], "all"))
	assert.False(t, HasExtension(byName["doc.pdf"], "txt"))
	assert.False(t, HasExtension(byName["noext"], "txt"))
}

// --- MatchesAnyExtension ---

func TestMatchesAnyExtension(t *testing.T) {
	assert.True(t, MatchesAnyExtension("file.txt", []string{"txt", "pdf"}))
	assert.True(t, MatchesAnyExtension("file.PDF", []string{"pdf"}))
	assert.True(t, MatchesAnyExtension("file.go", []string{"all"}))
	assert.False(t, MatchesAnyExtension("file.go", []string{"txt", "pdf"}))
}

// --- MatchesNameFilters ---

func TestMatchesNameFilters(t *testing.T) {
	t.Run("no filters passes all", func(t *testing.T) {
		assert.True(t, MatchesNameFilters("anything.txt", models.CategoryFilter{}))
	})

	t.Run("glob filter matches", func(t *testing.T) {
		f := models.CategoryFilter{Glob: "report_*"}
		assert.True(t, MatchesNameFilters("report_2024.pdf", f))
		assert.False(t, MatchesNameFilters("invoice.pdf", f))
	})

	t.Run("include filter", func(t *testing.T) {
		f := models.CategoryFilter{Include: []string{"*.pdf", "*.docx"}}
		assert.True(t, MatchesNameFilters("file.pdf", f))
		assert.False(t, MatchesNameFilters("file.txt", f))
	})
}

// --- ParseSize ---

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"100B", 100, false},
		{"1KB", 1024, false},
		{"1MB", 1 << 20, false},
		{"1GB", 1 << 30, false},
		{"1TB", 1 << 40, false},
		{"1.5MB", int64(1.5 * float64(1<<20)), false},
		{"500", 500, false},
		{"", 0, true},
		{"abcXB", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// --- Age/Size filters ---

func TestMeetsMinAge(t *testing.T) {
	dir := t.TempDir()
	path := createTempFile(t, dir, "old.txt")

	// backdate by 10 minutes
	old := time.Now().Add(-10 * time.Minute)
	require.NoError(t, os.Chtimes(path, old, old))

	info, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, MeetsMinAge(info, 0))
	assert.True(t, MeetsMinAge(info, 5*time.Minute))
	assert.False(t, MeetsMinAge(info, 20*time.Minute))
}

func TestMeetsMaxAge(t *testing.T) {
	dir := t.TempDir()
	path := createTempFile(t, dir, "recent.txt")

	old := time.Now().Add(-10 * time.Minute)
	require.NoError(t, os.Chtimes(path, old, old))

	info, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, MeetsMaxAge(info, 0))
	assert.True(t, MeetsMaxAge(info, 20*time.Minute))
	assert.False(t, MeetsMaxAge(info, 5*time.Minute))
}

func TestMeetsMinSize(t *testing.T) {
	dir := t.TempDir()
	path := createTempFile(t, dir, "data.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 500), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, MeetsMinSize(info, 0))
	assert.True(t, MeetsMinSize(info, 100))
	assert.False(t, MeetsMinSize(info, 1000))
}

func TestMeetsMaxSize(t *testing.T) {
	dir := t.TempDir()
	path := createTempFile(t, dir, "data.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 500), 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, MeetsMaxSize(info, 0))
	assert.True(t, MeetsMaxSize(info, 1000))
	assert.False(t, MeetsMaxSize(info, 100))
}

func TestMeetsAgeSizeFilters_NoConstraints(t *testing.T) {
	dir := t.TempDir()
	path := createTempFile(t, dir, "file.txt")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, MeetsAgeSizeFilters(info, models.CategoryFilter{}))
}

// createTempFile creates an empty file in dir and returns its path.
func createTempFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	return path
}
