package filters

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

func createTempFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	return path
}

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

func TestMeetsAge(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(os.FileInfo, time.Duration) bool
		fileAge   time.Duration
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

func makeInfo(t *testing.T, name string, size int, modTime time.Time) os.FileInfo {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content := make([]byte, size)
	require.NoError(t, os.WriteFile(path, content, 0644))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}

func TestMatchesFilter_Leaf(t *testing.T) {
	info := makeInfo(t, "report_2024.pdf", 1024, time.Now().Add(-2*time.Hour))

	tests := []struct {
		name   string
		filter models.CategoryFilter
		want   bool
	}{
		{"empty filter - no restrictions", models.CategoryFilter{}, true},
		{"glob matches", models.CategoryFilter{Glob: "report_*"}, true},
		{"glob no match", models.CategoryFilter{Glob: "invoice_*"}, false},
		{"ignore excludes file", models.CategoryFilter{Ignore: []string{"report_*"}}, false},
		{"min-size passes", models.CategoryFilter{MinSizeBytes: 512}, true},
		{"min-size fails", models.CategoryFilter{MinSizeBytes: 1024 * 1024}, false},
		{"min-age passes", models.CategoryFilter{MinAge: 1 * time.Hour}, true},
		{"min-age fails", models.CategoryFilter{MinAge: 3 * time.Hour}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchesFilter(tt.filter, info.Name(), info))
		})
	}
}

func TestMatchesFilter_Any(t *testing.T) {
	info := makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))

	t.Run("any - first group passes", func(t *testing.T) {
		f := models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Glob: "report_*"},
				{Glob: "invoice_*"},
			},
		}
		assert.True(t, MatchesFilter(f, info.Name(), info))
	})

	t.Run("any - second group passes", func(t *testing.T) {
		f := models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Glob: "invoice_*"},
				{Glob: "report_*"},
			},
		}
		assert.True(t, MatchesFilter(f, info.Name(), info))
	})

	t.Run("any - no group passes", func(t *testing.T) {
		f := models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Glob: "invoice_*"},
				{Glob: "draft_*"},
			},
		}
		assert.False(t, MatchesFilter(f, info.Name(), info))
	})
}

func TestMatchesFilter_All(t *testing.T) {
	info := makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))

	t.Run("all - all groups pass", func(t *testing.T) {
		f := models.CategoryFilter{
			All: []models.CategoryFilter{
				{Glob: "report_*"},
				{MinSizeBytes: 1024 * 1024},
			},
		}
		assert.True(t, MatchesFilter(f, info.Name(), info))
	})

	t.Run("all - one group fails", func(t *testing.T) {
		f := models.CategoryFilter{
			All: []models.CategoryFilter{
				{Glob: "report_*"},
				{MinSizeBytes: 10 * 1024 * 1024},
			},
		}
		assert.False(t, MatchesFilter(f, info.Name(), info))
	})
}

func TestMatchesFilter_AnyInsideAll(t *testing.T) {
	info := makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now())

	f := models.CategoryFilter{
		All: []models.CategoryFilter{
			{MinSizeBytes: 1024 * 1024},
			{
				Any: []models.CategoryFilter{
					{Glob: "report_*"},
					{Glob: "invoice_*"},
				},
			},
		},
	}
	assert.True(t, MatchesFilter(f, info.Name(), info))

	smallInfo := makeInfo(t, "report_small.pdf", 512, time.Now())
	assert.False(t, MatchesFilter(f, smallInfo.Name(), smallInfo))
}

func TestMatchesFilter_AllInsideAny(t *testing.T) {
	f := models.CategoryFilter{
		Any: []models.CategoryFilter{
			{
				All: []models.CategoryFilter{
					{Glob: "report_*"},
					{MinSizeBytes: 1024 * 1024},
				},
			},
			{
				All: []models.CategoryFilter{
					{Glob: "invoice_*"},
					{MinAge: 2 * time.Hour},
				},
			},
		},
	}

	reportLarge := makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now())
	assert.True(t, MatchesFilter(f, reportLarge.Name(), reportLarge))

	invoiceOld := makeInfo(t, "invoice_jan.pdf", 100, time.Now().Add(-3*time.Hour))
	assert.True(t, MatchesFilter(f, invoiceOld.Name(), invoiceOld))

	reportSmall := makeInfo(t, "report_tiny.pdf", 100, time.Now())
	assert.False(t, MatchesFilter(f, reportSmall.Name(), reportSmall))
}

func TestMeetsAgeSizeFilters(t *testing.T) {
	tests := []struct {
		name     string
		fileAge  time.Duration
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

func TestGenerateLogArgs(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		ext     string
		wantLen int
	}{
		{"matches by extension", []string{"a.pdf", "b.pdf", "c.txt"}, "pdf", 4},
		{"no match returns empty", []string{"file.txt"}, "pdf", 0},
		{"all extension matches everything", []string{"a.pdf", "b.txt"}, "all", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644))
			}
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)

			args := GenerateLogArgs(entries, tt.ext)
			assert.Len(t, args, tt.wantLen)
			for i := 0; i < len(args)-1; i += 2 {
				assert.Equal(t, "name", args[i])
			}
		})
	}
}
