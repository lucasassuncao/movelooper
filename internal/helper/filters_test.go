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

// createTempFile creates a file with default content in dir and returns its path.
func createTempFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	return path
}

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
			assert.Equal(t, tt.want, MatchesIgnorePatterns(tt.fileName, tt.patterns, tt.caseSensitive))
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
		{"*.{ jpg , png }", []string{"*.jpg", "*.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			assert.Equal(t, tt.want, expandGlobPattern(tt.pattern))
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
			assert.Equal(t, tt.want, MatchesGlob(tt.fileName, tt.pattern, tt.caseSensitive))
		})
	}
}

// --- ValidateGlob ---

func TestValidateGlob(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"*.txt", false},
		{"*.{jpg,png}", false},
		{"[invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := ValidateGlob(tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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

	tests := []struct {
		name  string
		entry os.DirEntry
		ext   string
		want  bool
	}{
		{"exact match", byName["doc.pdf"], "pdf", true},
		{"case insensitive", byName["image.PNG"], "png", true},
		{"all matches any", byName["doc.pdf"], "all", true},
		{"wrong ext", byName["doc.pdf"], "txt", false},
		{"no ext vs txt", byName["noext"], "txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HasExtension(tt.entry, tt.ext))
		})
	}
}

// --- MatchesAnyExtension ---

func TestMatchesAnyExtension(t *testing.T) {
	tests := []struct {
		name string
		file string
		exts []string
		want bool
	}{
		{"matches first ext", "file.txt", []string{"txt", "pdf"}, true},
		{"case insensitive", "file.PDF", []string{"pdf"}, true},
		{"all matches any", "file.go", []string{"all"}, true},
		{"no match", "file.go", []string{"txt", "pdf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchesAnyExtension(tt.file, tt.exts))
		})
	}
}

// --- MatchesNameFilters ---

func TestMatchesNameFilters(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		filter   models.CategoryFilter
		want     bool
	}{
		{"no filters passes all", "anything.txt", models.CategoryFilter{}, true},
		{"glob matches", "report_2024.pdf", models.CategoryFilter{Glob: "report_*"}, true},
		{"glob no match", "invoice.pdf", models.CategoryFilter{Glob: "report_*"}, false},
		{"include matches first", "file.pdf", models.CategoryFilter{Include: []string{"*.pdf", "*.docx"}}, true},
		{"include no match", "file.txt", models.CategoryFilter{Include: []string{"*.pdf", "*.docx"}}, false},
		{"include multiple patterns first", "IMG_001.jpg", models.CategoryFilter{Include: []string{"IMG_*", "DSC_*"}}, true},
		{"include multiple patterns second", "DSC_100.jpg", models.CategoryFilter{Include: []string{"IMG_*", "DSC_*"}}, true},
		{"include no match multiple", "photo.jpg", models.CategoryFilter{Include: []string{"IMG_*", "DSC_*"}}, false},
		{
			"regex match",
			"report_2024.pdf",
			models.CategoryFilter{Regex: "report", CompiledRegex: regexp.MustCompile("(?i)report")},
			true,
		},
		{
			"regex no match",
			"invoice.pdf",
			models.CategoryFilter{Regex: "^report", CompiledRegex: regexp.MustCompile("^report")},
			false,
		},
		{
			"glob filter match",
			"report_2024.pdf",
			models.CategoryFilter{Glob: "report_*.pdf"},
			true,
		},
		{
			"glob filter no match",
			"invoice.pdf",
			models.CategoryFilter{Glob: "report_*.pdf"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchesNameFilters(tt.fileName, tt.filter))
		})
	}
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

// --- MeetsMinAge / MeetsMaxAge ---

func TestMeetsAge(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(os.FileInfo, time.Duration) bool
		fileAge   time.Duration // how old to backdate the file
		threshold time.Duration
		want      bool
	}{
		{"min: zero threshold always passes", MeetsMinAge, 10 * time.Minute, 0, true},
		{"min: file older than threshold", MeetsMinAge, 10 * time.Minute, 5 * time.Minute, true},
		{"min: file newer than threshold", MeetsMinAge, 10 * time.Minute, 20 * time.Minute, false},
		{"max: zero threshold always passes", MeetsMaxAge, 10 * time.Minute, 0, true},
		{"max: file within threshold", MeetsMaxAge, 10 * time.Minute, 20 * time.Minute, true},
		{"max: file exceeds threshold", MeetsMaxAge, 10 * time.Minute, 5 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTempFile(t, t.TempDir(), "file.txt")
			ts := time.Now().Add(-tt.fileAge)
			require.NoError(t, os.Chtimes(path, ts, ts))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.fn(info, tt.threshold))
		})
	}
}

// --- MeetsMinSize / MeetsMaxSize ---

func TestMeetsSize(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(os.FileInfo, int64) bool
		fileSize  int
		threshold int64
		want      bool
	}{
		{"min: zero threshold always passes", MeetsMinSize, 500, 0, true},
		{"min: file bigger than threshold", MeetsMinSize, 500, 100, true},
		{"min: file smaller than threshold", MeetsMinSize, 500, 1000, false},
		{"max: zero threshold always passes", MeetsMaxSize, 500, 0, true},
		{"max: file within threshold", MeetsMaxSize, 500, 1000, true},
		{"max: file exceeds threshold", MeetsMaxSize, 500, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTempFile(t, t.TempDir(), "file.bin")
			require.NoError(t, os.WriteFile(path, make([]byte, tt.fileSize), 0644))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.fn(info, tt.threshold))
		})
	}
}

// --- MeetsAgeSizeFilters ---

func TestMeetsAgeSizeFilters(t *testing.T) {
	tests := []struct {
		name     string
		fileAge  time.Duration // how old to backdate; zero = fresh
		fileSize int
		filter   models.CategoryFilter
		want     bool
	}{
		{"no constraints passes", 0, 4, models.CategoryFilter{}, true},
		{"min-age only: passes", 2 * time.Hour, 1, models.CategoryFilter{MinAge: 1 * time.Hour}, true},
		{"min-age only: fails", 0, 1, models.CategoryFilter{MinAge: 1 * time.Hour}, false},
		{"max-age only: passes", 0, 1, models.CategoryFilter{MaxAge: 1 * time.Hour}, true},
		{"max-age only: fails", 2 * time.Hour, 1, models.CategoryFilter{MaxAge: 1 * time.Hour}, false},
		{"min-size only: passes", 0, 2048, models.CategoryFilter{MinSizeBytes: 1024}, true},
		{"min-size only: fails", 0, 4, models.CategoryFilter{MinSizeBytes: 1024}, false},
		{"max-size only: passes", 0, 4, models.CategoryFilter{MaxSizeBytes: 1024}, true},
		{"max-size only: fails", 0, 2048, models.CategoryFilter{MaxSizeBytes: 1024}, false},
		{
			"all constraints pass",
			2 * time.Hour, 512,
			models.CategoryFilter{MinAge: 1 * time.Hour, MaxAge: 24 * time.Hour, MinSizeBytes: 100, MaxSizeBytes: 1024},
			true,
		},
		{
			"all constraints: size fails",
			2 * time.Hour, 2048,
			models.CategoryFilter{MinAge: 1 * time.Hour, MaxAge: 24 * time.Hour, MinSizeBytes: 100, MaxSizeBytes: 1024},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTempFile(t, t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(path, make([]byte, tt.fileSize), 0644))
			if tt.fileAge > 0 {
				ts := time.Now().Add(-tt.fileAge)
				require.NoError(t, os.Chtimes(path, ts, ts))
			}
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, MeetsAgeSizeFilters(info, tt.filter))
		})
	}
}
