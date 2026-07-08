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
	require.NoError(t, os.WriteFile(path, []byte("test"), 0o644))
	return path
}

func makeInfo(t *testing.T, name string, size int, modTime time.Time) os.FileInfo {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content := make([]byte, size)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}

// testExpandGlobPattern defines the structure for test cases of the expandGlobPattern function,
// containing the input pattern and the expected expanded patterns.
type testExpandGlobPattern struct {
	pattern string
	want    []string
}

// testExpandGlobPatternTestCases defines a set of test cases for the expandGlobPattern function,
// covering simple patterns, brace expansion, and whitespace trimming.
var testExpandGlobPatternTestCases = []testExpandGlobPattern{
	{"*.txt", []string{"*.txt"}},
	{"*.{jpg,png}", []string{"*.jpg", "*.png"}},
	{"file.{go,py,js}", []string{"file.go", "file.py", "file.js"}},
	{"{a,b}", []string{"a", "b"}},
	{"no braces", []string{"no braces"}},
	{"*.{ jpg , png }", []string{"*.jpg", "*.png"}},
	{"{a,b}/{c,d}", []string{"a/c", "a/d", "b/c", "b/d"}},
	// A literal "}" before a valid group must not hide that group: the search
	// for the matching "}" must start after the "{", not from the beginning.
	{"a}b{c,d}", []string{"a}bc", "a}bd"}},
}

// TestExpandGlobPattern tests the expandGlobPattern function to ensure it correctly expands brace patterns.
func TestExpandGlobPattern(t *testing.T) {
	t.Parallel()
	for _, tt := range testExpandGlobPatternTestCases {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, expandGlobPattern(tt.pattern))
		})
	}
}

// testMatchesGlob defines the structure for test cases of the MatchesGlob function,
// containing the file name, glob pattern, case sensitivity flag, and expected result.
type testMatchesGlob struct {
	name          string
	fileName      string
	pattern       string
	caseSensitive bool
	want          bool
}

// testMatchesGlobTestCases defines a set of test cases for the MatchesGlob function,
// covering simple match, brace expansion, case sensitivity, and wildcard name patterns.
var testMatchesGlobTestCases = []testMatchesGlob{
	{"simple match", "photo.jpg", "*.jpg", false, true},
	{"brace expansion match", "photo.jpg", "*.{jpg,png}", false, true},
	{"brace expansion second", "photo.png", "*.{jpg,png}", false, true},
	{"no match", "photo.gif", "*.{jpg,png}", false, false},
	{"case insensitive", "PHOTO.JPG", "*.jpg", false, true},
	{"case sensitive no match", "PHOTO.JPG", "*.jpg", true, false},
	{"wildcard name", "report_2024.pdf", "report_*.pdf", false, true},
}

// TestMatchesGlob tests the MatchesGlob function to ensure it correctly matches file names against glob patterns.
func TestMatchesGlob(t *testing.T) {
	t.Parallel()
	for _, tt := range testMatchesGlobTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MatchesGlob(tt.fileName, tt.pattern, tt.caseSensitive))
		})
	}
}

// testValidateGlob defines the structure for test cases of the ValidateGlob function,
// containing the glob pattern and an error expectation flag.
type testValidateGlob struct {
	pattern string
	wantErr bool
}

// testValidateGlobTestCases defines a set of test cases for the ValidateGlob function,
// covering valid patterns, brace expansion, and invalid patterns.
var testValidateGlobTestCases = []testValidateGlob{
	{"*.txt", false},
	{"*.{jpg,png}", false},
	{"[invalid", true},
}

// TestValidateGlob tests the ValidateGlob function to ensure it correctly validates glob patterns.
func TestValidateGlob(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidateGlobTestCases {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()
			err := ValidateGlob(tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testHasExtension defines the structure for test cases of the HasExtension function,
// containing the file name to look up, the extension to check, and the expected result.
type testHasExtension struct {
	name     string
	fileName string
	ext      string
	want     bool
}

// testHasExtensionTestCases defines a set of test cases for the HasExtension function,
// covering exact match, case insensitivity, all extension, wrong extension, and no extension.
var testHasExtensionTestCases = []testHasExtension{
	{"exact match", "doc.pdf", "pdf", true},
	{"case insensitive", "image.PNG", "png", true},
	{"all matches any", "doc.pdf", "all", true},
	{"wrong ext", "doc.pdf", "txt", false},
	{"no ext vs txt", "noext", "txt", false},
}

// TestHasExtension tests the HasExtension function to ensure it correctly identifies file extensions.
func TestHasExtension(t *testing.T) {
	t.Parallel()
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

	for _, tt := range testHasExtensionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasExtension(byName[tt.fileName], tt.ext))
		})
	}
}

// testMatchesAnyExtension defines the structure for test cases of the MatchesAnyExtension function,
// containing the file name, list of extensions, and expected result.
type testMatchesAnyExtension struct {
	name string
	file string
	exts []string
	want bool
}

// testMatchesAnyExtensionTestCases defines a set of test cases for the MatchesAnyExtension function,
// covering first ext match, case insensitivity, all extension, and no match.
var testMatchesAnyExtensionTestCases = []testMatchesAnyExtension{
	{"matches first ext", "file.txt", []string{"txt", "pdf"}, true},
	{"case insensitive", "file.PDF", []string{"pdf"}, true},
	{"all matches any", "file.go", []string{"all"}, true},
	{"no match", "file.go", []string{"txt", "pdf"}, false},
}

// TestMatchesAnyExtension tests the MatchesAnyExtension function to ensure it correctly
// matches file names against a list of extensions.
func TestMatchesAnyExtension(t *testing.T) {
	t.Parallel()
	for _, tt := range testMatchesAnyExtensionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MatchesAnyExtension(tt.file, tt.exts))
		})
	}
}

// testMatchesNameFilters defines the structure for test cases of the MatchesNameFilters function,
// containing the file name, the category filter, and the expected result.
type testMatchesNameFilters struct {
	name     string
	fileName string
	filter   models.CategoryFilter
	want     bool
}

// testMatchesNameFiltersTestCases defines a set of test cases for the MatchesNameFilters function,
// covering no filters, glob, literal, and regex scenarios.
var testMatchesNameFiltersTestCases = []testMatchesNameFilters{
	{"no filters passes all", "anything.txt", models.CategoryFilter{}, true},
	{"glob matches", "report_2024.pdf", models.CategoryFilter{Match: &models.MatchFilter{Glob: "report_*"}}, true},
	{"glob no match", "invoice.pdf", models.CategoryFilter{Match: &models.MatchFilter{Glob: "report_*"}}, false},
	{"literal match", "report.pdf", models.CategoryFilter{Match: &models.MatchFilter{Literal: "report.pdf"}}, true},
	{"literal no match", "REPORT.PDF", models.CategoryFilter{Match: &models.MatchFilter{Literal: "report.pdf", CaseSensitive: true}}, false},
	{"literal case insensitive", "REPORT.PDF", models.CategoryFilter{Match: &models.MatchFilter{Literal: "report.pdf", CaseSensitive: false}}, true},
	{
		"regex match",
		"report_2024.pdf",
		models.CategoryFilter{Match: &models.MatchFilter{
			Regex:         "report",
			CompiledRegex: regexp.MustCompile("(?i)report"),
		}},
		true,
	},
	{
		"regex no match",
		"invoice.pdf",
		models.CategoryFilter{Match: &models.MatchFilter{
			Regex:         "^report",
			CompiledRegex: regexp.MustCompile("^report"),
		}},
		false,
	},
	{"glob filter match", "report_2024.pdf", models.CategoryFilter{Match: &models.MatchFilter{Glob: "report_*.pdf"}}, true},
	{"glob filter no match", "invoice.pdf", models.CategoryFilter{Match: &models.MatchFilter{Glob: "report_*.pdf"}}, false},
}

// TestMatchesNameFilters tests the MatchesNameFilters function to ensure it correctly
// applies name-based filters to file names.
func TestMatchesNameFilters(t *testing.T) {
	t.Parallel()
	for _, tt := range testMatchesNameFiltersTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MatchesNameFilters(tt.fileName, tt.filter))
		})
	}
}

// testParseSize defines the structure for test cases of the ParseSize function,
// containing the input string, expected byte count, and an error expectation flag.
type testParseSize struct {
	input   string
	want    int64
	wantErr bool
}

// testParseSizeTestCases defines a set of test cases for the ParseSize function,
// covering bytes, decimal (KB/MB/GB/TB) and binary (KiB/MiB/GiB/TiB) units,
// decimals, bare numbers, and invalid input.
var testParseSizeTestCases = []testParseSize{
	{"100B", 100, false},
	{"1KB", 1_000, false},
	{"1MB", 1_000_000, false},
	{"1GB", 1_000_000_000, false},
	{"1TB", 1_000_000_000_000, false},
	{"1KiB", 1 << 10, false},
	{"1MiB", 1 << 20, false},
	{"1GiB", 1 << 30, false},
	{"1TiB", 1 << 40, false},
	{"1.5MB", 1_500_000, false},
	{"1.5MiB", int64(1.5 * float64(1<<20)), false},
	{"500", 500, false},
	{"", 0, true},
	{"abcXB", 0, true},
	{"-5MB", 0, true},
	{"-100", 0, true},
	{"1000000000000TB", 0, true},
	{"12abc", 0, true},
	{"1.5.2GB", 0, true},
	{"10 MB", 10_000_000, false},
}

// TestParseSize tests the ParseSize function to ensure it correctly parses human-readable size strings.
func TestParseSize(t *testing.T) {
	t.Parallel()
	for _, tt := range testParseSizeTestCases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
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

// testMeetsAge defines the structure for test cases of the MeetsMinAge and MeetsMaxAge functions,
// containing the function under test, file age, threshold, and expected result.
type testMeetsAge struct {
	name      string
	fn        func(os.FileInfo, time.Duration) bool
	fileAge   time.Duration
	threshold time.Duration
	want      bool
}

// testMeetsAgeTestCases defines a set of test cases for the MeetsMinAge and MeetsMaxAge functions,
// covering zero threshold, file older/newer than threshold, and max age scenarios.
var testMeetsAgeTestCases = []testMeetsAge{
	{"min: zero threshold always passes", MeetsMinAge, 10 * time.Minute, 0, true},
	{"min: file older than threshold", MeetsMinAge, 10 * time.Minute, 5 * time.Minute, true},
	{"min: file newer than threshold", MeetsMinAge, 10 * time.Minute, 20 * time.Minute, false},
	{"max: zero threshold always passes", MeetsMaxAge, 10 * time.Minute, 0, true},
	{"max: file within threshold", MeetsMaxAge, 10 * time.Minute, 20 * time.Minute, true},
	{"max: file exceeds threshold", MeetsMaxAge, 10 * time.Minute, 5 * time.Minute, false},
}

// TestMeetsAge tests the MeetsMinAge and MeetsMaxAge functions to ensure they correctly evaluate file age.
func TestMeetsAge(t *testing.T) {
	t.Parallel()
	for _, tt := range testMeetsAgeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := createTempFile(t, t.TempDir(), "file.txt")
			ts := time.Now().Add(-tt.fileAge)
			require.NoError(t, os.Chtimes(path, ts, ts))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.fn(info, tt.threshold))
		})
	}
}

// testMeetsSize defines the structure for test cases of the MeetsMinSize and MeetsMaxSize functions,
// containing the function under test, file size, threshold, and expected result.
type testMeetsSize struct {
	name      string
	fn        func(os.FileInfo, int64) bool
	fileSize  int
	threshold int64
	want      bool
}

// testMeetsSizeTestCases defines a set of test cases for the MeetsMinSize and MeetsMaxSize functions,
// covering zero threshold, file bigger/smaller than threshold, and max size scenarios.
var testMeetsSizeTestCases = []testMeetsSize{
	{"min: zero threshold always passes", MeetsMinSize, 500, 0, true},
	{"min: file bigger than threshold", MeetsMinSize, 500, 100, true},
	{"min: file smaller than threshold", MeetsMinSize, 500, 1000, false},
	{"max: zero threshold always passes", MeetsMaxSize, 500, 0, true},
	{"max: file within threshold", MeetsMaxSize, 500, 1000, true},
	{"max: file exceeds threshold", MeetsMaxSize, 500, 100, false},
}

// TestMeetsSize tests the MeetsMinSize and MeetsMaxSize functions to ensure they correctly evaluate file size.
func TestMeetsSize(t *testing.T) {
	t.Parallel()
	for _, tt := range testMeetsSizeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := createTempFile(t, t.TempDir(), "file.bin")
			require.NoError(t, os.WriteFile(path, make([]byte, tt.fileSize), 0o644))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.fn(info, tt.threshold))
		})
	}
}

// testMatchesFilterLeaf defines the structure for test cases of the MatchesFilter function
// with leaf (non-composite) filters, containing the filter and expected result.
type testMatchesFilterLeaf struct {
	name   string
	filter models.CategoryFilter
	want   bool
}

// testMatchesFilterLeafTestCases defines a set of test cases for the MatchesFilter function
// with leaf filters, covering empty filter, glob, not, min-size, and min-age scenarios.
var testMatchesFilterLeafTestCases = []testMatchesFilterLeaf{
	{"empty filter - no restrictions", models.CategoryFilter{}, true},
	{"glob matches", models.CategoryFilter{Match: &models.MatchFilter{Glob: "report_*"}}, true},
	{"glob no match", models.CategoryFilter{Match: &models.MatchFilter{Glob: "invoice_*"}}, false},
	{"not excludes file", models.CategoryFilter{Not: []models.CategoryFilter{
		{Match: &models.MatchFilter{Glob: "report_*"}},
	}}, false},
	{"min-size passes", models.CategoryFilter{Size: &models.SizeFilter{MinBytes: 512}}, true},
	{"min-size fails", models.CategoryFilter{Size: &models.SizeFilter{MinBytes: 1024 * 1024}}, false},
	{"min-age passes", models.CategoryFilter{Age: &models.AgeFilter{Min: 1 * time.Hour}}, true},
	{"min-age fails", models.CategoryFilter{Age: &models.AgeFilter{Min: 3 * time.Hour}}, false},
}

// TestMatchesFilter_Leaf tests the MatchesFilter function with leaf filters
// to ensure it correctly evaluates individual filter conditions.
func TestMatchesFilter_Leaf(t *testing.T) {
	t.Parallel()
	info := makeInfo(t, "report_2024.pdf", 1024, time.Now().Add(-2*time.Hour))
	for _, tt := range testMatchesFilterLeafTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MatchesFilter(tt.filter, info.Name(), info))
		})
	}
}

// testMatchesFilterComposite defines the structure for composite (Any/All) filter test cases,
// containing the filter, a file info builder, and the expected result.
type testMatchesFilterComposite struct {
	name   string
	filter models.CategoryFilter
	info   func(t *testing.T) os.FileInfo
	want   bool
}

// testMatchesFilterCompositeTestCases defines a set of test cases for the MatchesFilter function
// with Any, All, and Not composite filters, including nested combinations.
var testMatchesFilterCompositeTestCases = []testMatchesFilterComposite{
	{
		name: "any - first group passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{Match: &models.MatchFilter{Glob: "report_*"}},
			{Match: &models.MatchFilter{Glob: "invoice_*"}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))
		},
		want: true,
	},
	{
		name: "any - second group passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{Match: &models.MatchFilter{Glob: "invoice_*"}},
			{Match: &models.MatchFilter{Glob: "report_*"}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))
		},
		want: true,
	},
	{
		name: "any - no group passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{Match: &models.MatchFilter{Glob: "invoice_*"}},
			{Match: &models.MatchFilter{Glob: "draft_*"}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))
		},
		want: false,
	},
	{
		name: "all - all groups pass",
		filter: models.CategoryFilter{All: []models.CategoryFilter{
			{Match: &models.MatchFilter{Glob: "report_*"}},
			{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))
		},
		want: true,
	},
	{
		name: "all - one group fails",
		filter: models.CategoryFilter{All: []models.CategoryFilter{
			{Match: &models.MatchFilter{Glob: "report_*"}},
			{Size: &models.SizeFilter{MinBytes: 10 * 1024 * 1024}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now().Add(-2*time.Hour))
		},
		want: false,
	},
	{
		name: "any inside all - passes",
		filter: models.CategoryFilter{All: []models.CategoryFilter{
			{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
			{Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
			}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now())
		},
		want: true,
	},
	{
		name: "any inside all - size fails",
		filter: models.CategoryFilter{All: []models.CategoryFilter{
			{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
			{Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
			}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_small.pdf", 512, time.Now())
		},
		want: false,
	},
	{
		name: "all inside any - first branch passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
			}},
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
				{Age: &models.AgeFilter{Min: 2 * time.Hour}},
			}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 2*1024*1024, time.Now())
		},
		want: true,
	},
	{
		name: "all inside any - second branch passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
			}},
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
				{Age: &models.AgeFilter{Min: 2 * time.Hour}},
			}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "invoice_jan.pdf", 100, time.Now().Add(-3*time.Hour))
		},
		want: true,
	},
	{
		name: "all inside any - no branch passes",
		filter: models.CategoryFilter{Any: []models.CategoryFilter{
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Size: &models.SizeFilter{MinBytes: 1024 * 1024}},
			}},
			{All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
				{Age: &models.AgeFilter{Min: 2 * time.Hour}},
			}},
		}},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_tiny.pdf", 100, time.Now())
		},
		want: false,
	},
	{
		name: "not excludes matching file",
		filter: models.CategoryFilter{
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "draft_*"}},
			},
		},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "draft_2024.pdf", 100, time.Now())
		},
		want: false,
	},
	{
		name: "not passes non-matching file",
		filter: models.CategoryFilter{
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "draft_*"}},
			},
		},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 100, time.Now())
		},
		want: true,
	},
	{
		name: "not alongside any - excludes matching draft",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
			},
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "*draft*"}},
			},
		},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_draft.pdf", 100, time.Now())
		},
		want: false,
	},
	{
		name: "not alongside any - keeps non-draft",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
			},
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "*draft*"}},
			},
		},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_2024.pdf", 100, time.Now())
		},
		want: true,
	},
	{
		name: "not alongside all - excludes matching draft",
		filter: models.CategoryFilter{
			All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Size: &models.SizeFilter{MinBytes: 100}},
			},
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "*draft*"}},
			},
		},
		info: func(t *testing.T) os.FileInfo {
			return makeInfo(t, "report_draft.pdf", 2048, time.Now())
		},
		want: false,
	},
}

// TestMatchesFilter_Composite tests the MatchesFilter function with composite Any/All filters
// to ensure it correctly evaluates nested filter conditions.
func TestMatchesFilter_Composite(t *testing.T) {
	t.Parallel()
	for _, tt := range testMatchesFilterCompositeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := tt.info(t)
			assert.Equal(t, tt.want, MatchesFilter(tt.filter, info.Name(), info))
		})
	}
}

// testMeetsAgeSizeFilters defines the structure for test cases of the MeetsAgeSizeFilters function,
// containing the file age, file size, filter, and expected result.
type testMeetsAgeSizeFilters struct {
	name     string
	fileAge  time.Duration
	fileSize int
	filter   models.CategoryFilter
	want     bool
}

// testMeetsAgeSizeFiltersTestCases defines a set of test cases for the MeetsAgeSizeFilters function,
// covering no constraints, individual age and size filters, and all constraints combined.
var testMeetsAgeSizeFiltersTestCases = []testMeetsAgeSizeFilters{
	{"no constraints passes", 0, 4, models.CategoryFilter{}, true},
	{"min-age only: passes", 2 * time.Hour, 1, models.CategoryFilter{Age: &models.AgeFilter{Min: 1 * time.Hour}}, true},
	{"min-age only: fails", 0, 1, models.CategoryFilter{Age: &models.AgeFilter{Min: 1 * time.Hour}}, false},
	{"max-age only: passes", 0, 1, models.CategoryFilter{Age: &models.AgeFilter{Max: 1 * time.Hour}}, true},
	{"max-age only: fails", 2 * time.Hour, 1, models.CategoryFilter{Age: &models.AgeFilter{Max: 1 * time.Hour}}, false},
	{"min-size only: passes", 0, 2048, models.CategoryFilter{Size: &models.SizeFilter{MinBytes: 1024}}, true},
	{"min-size only: fails", 0, 4, models.CategoryFilter{Size: &models.SizeFilter{MinBytes: 1024}}, false},
	{"max-size only: passes", 0, 4, models.CategoryFilter{Size: &models.SizeFilter{MaxBytes: 1024}}, true},
	{"max-size only: fails", 0, 2048, models.CategoryFilter{Size: &models.SizeFilter{MaxBytes: 1024}}, false},
	{
		"all constraints pass",
		2 * time.Hour, 512,
		models.CategoryFilter{
			Age:  &models.AgeFilter{Min: 1 * time.Hour, Max: 24 * time.Hour},
			Size: &models.SizeFilter{MinBytes: 100, MaxBytes: 1024},
		},
		true,
	},
	{
		"all constraints: size fails",
		2 * time.Hour, 2048,
		models.CategoryFilter{
			Age:  &models.AgeFilter{Min: 1 * time.Hour, Max: 24 * time.Hour},
			Size: &models.SizeFilter{MinBytes: 100, MaxBytes: 1024},
		},
		false,
	},
}

// TestMeetsAgeSizeFilters tests the MeetsAgeSizeFilters function to ensure it correctly
// evaluates combined age and size constraints.
func TestMeetsAgeSizeFilters(t *testing.T) {
	t.Parallel()
	for _, tt := range testMeetsAgeSizeFiltersTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := createTempFile(t, t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(path, make([]byte, tt.fileSize), 0o644))
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

// testGenerateLogArgs defines the structure for test cases of the GenerateLogArgs function,
// containing the files to create, the extension to filter by, and the expected argument count.
type testGenerateLogArgs struct {
	name    string
	files   []string
	ext     string
	wantLen int
}

// testGenerateLogArgsTestCases defines a set of test cases for the GenerateLogArgs function,
// covering extension match, no match, and all extension scenarios.
var testGenerateLogArgsTestCases = []testGenerateLogArgs{
	{"matches by extension", []string{"a.pdf", "b.pdf", "c.txt"}, "pdf", 4},
	{"no match returns empty", []string{"file.txt"}, "pdf", 0},
	{"all extension matches everything", []string{"a.pdf", "b.txt"}, "all", 4},
}

// TestGenerateLogArgs tests the GenerateLogArgs function to ensure it correctly
// generates log argument pairs for matching files.
func TestGenerateLogArgs(t *testing.T) {
	t.Parallel()
	for _, tt := range testGenerateLogArgsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
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

func TestMatchesFilter_Mime(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	png := filepath.Join(dir, "photo.jpg") // wrong extension on purpose
	require.NoError(t, os.WriteFile(png, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644))
	info, err := os.Stat(png)
	require.NoError(t, err)

	assert.True(t, MatchesFilter(models.CategoryFilter{Mime: "image/*"}, png, info), "PNG content matches image/*")
	assert.True(t, MatchesFilter(models.CategoryFilter{Mime: "image/png"}, png, info))
	assert.False(t, MatchesFilter(models.CategoryFilter{Mime: "text/*"}, png, info))
	assert.False(t, MatchesFilter(models.CategoryFilter{Not: []models.CategoryFilter{{Mime: "image/*"}}}, png, info), "not image/* excludes a PNG")

	missing := filepath.Join(dir, "gone.bin")
	assert.False(t, MatchesFilter(models.CategoryFilter{Mime: "image/*"}, missing, info), "unreadable file does not match a positive mime rule")
}
