package tokens

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testResolveRename defines the structure for test cases of the ResolveRename function.
type testResolveRename struct {
	name         string
	template     string
	want         string
	emptyDestDir bool
}

// testResolveRenameTestCases contains various scenarios to test the ResolveRename function, including edge cases and typical use cases.
var testResolveRenameTestCases = []testResolveRename{
	{"empty template returns original name", "", "photo.JPG", false},
	{"ext token lowercase", "{name}.{ext}", "photo.jpg", false},
	{"ext-upper token", "{name}.{ext-upper}", "photo.JPG", false},
	{"mod-date prefix", "{mod-date}_{name}.{ext}", "2024-03-05_photo.jpg", false},
	{"category token", "{category}_{name}.{ext}", "images_photo.jpg", false},
	{"run date", "{date}_{name}.{ext}", "2025-04-16_photo.jpg", false},
	{"seq no padding", "{seq}_{name}.{ext}", "1_photo.jpg", false},
	{"seq with padding 3", "{seq:3}_{name}.{ext}", "001_photo.jpg", false},
	{"seq with padding 4", "{seq:4}_{name}.{ext}", "0001_photo.jpg", false},
	{"seq-alpha empty dir", "{seq-alpha}_{name}.{ext}", "a_photo.jpg", false},
	{"seq-roman empty dir", "{seq-roman}_{name}.{ext}", "i_photo.jpg", false},
	// md5 of "x" = 9dd4e461268c8034f5c8564e155c67a6
	{"md5 default 8 chars", "{md5}_{name}.{ext}", "9dd4e461_photo.jpg", false},
	{"md5:4", "{md5:4}_{name}.{ext}", "9dd4_photo.jpg", false},
	// sha256 of "x" = 2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881
	{"sha256:6", "{sha256:6}_{name}.{ext}", "2d7116_photo.jpg", false},
	// path separator in result is replaced with underscore
	{"path separator stripped", "{category}/{name}.{ext}", "images_photo.jpg", false},
	// emptyDestDir=true exercises the else branch in ResolveRename (no lock acquired)
	{"empty destDir seq falls back to 1", "{seq}_{name}.{ext}", "1_photo.jpg", true},
}

// TestResolveRename iterates through the test cases defined in testResolveRenameTestCases, sets up the necessary context for each case, and asserts that the output of ResolveRename matches the expected result.
func TestResolveRename(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 16, 0, 0, 0, 0, time.UTC)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "photo.JPG")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	modTime := time.Date(2024, 3, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(path, modTime, modTime))
	info, err := os.Stat(path)
	require.NoError(t, err)

	destDir := t.TempDir()

	for _, tt := range testResolveRenameTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := TokenContext{
				Info:         info,
				CategoryName: "images",
				Now:          now,
				SourcePath:   path,
			}
			if !tt.emptyDestDir {
				ctx.DestDir = destDir
			}
			got := ResolveRename(tt.template, &ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// testResolveGroupBy defines the structure for test cases of the ResolveGroupBy function.
type testResolveGroupBy struct {
	name     string
	template string
	info     os.FileInfo
	category string
	now      time.Time
	want     string
}

// testResolveGroupByTestCases builds the test cases at runtime because they depend on
// os.FileInfo and createdTime values that are only available after file fixtures are created.
func testResolveGroupByTestCases(plain, tiny, mid os.FileInfo, createdTime, now, modTime time.Time) []testResolveGroupBy {
	return []testResolveGroupBy{
		{"empty template", "", plain, "docs", now, ""},
		// identification
		{"name", "{name}", plain, "docs", now, "my-report"},
		{"ext lowercase", "{ext}", plain, "docs", now, "pdf"},
		{"ext uppercase", "{ext-upper}", plain, "docs", now, "PDF"},
		{"ext-lower", "{ext-lower}", plain, "docs", now, "pdf"},
		{"ext-reverse", "{ext-reverse}", plain, "docs", now, "fdp"},
		// name transforms
		{"name-slug", "{name-slug}", plain, "docs", now, "my-report"},
		{"name-snake", "{name-snake}", plain, "docs", now, "my_report"},
		{"name-upper", "{name-upper}", plain, "docs", now, "MY-REPORT"},
		{"name-lower", "{name-lower}", plain, "docs", now, "my-report"},
		{"name-alpha", "{name-alpha}", plain, "docs", now, "myreport"},
		{"name-ascii", "{name-ascii}", plain, "docs", now, "my-report"},
		{"name-initials", "{name-initials}", plain, "docs", now, "mr"},
		{"name-reverse", "{name-reverse}", plain, "docs", now, "troper-ym"},
		// modification date
		{"mod-year", "{mod-year}", plain, "cat", now, "2023"},
		{"mod-month", "{mod-month}", plain, "cat", now, "07"},
		{"mod-day", "{mod-day}", plain, "cat", now, "04"},
		{"mod-date", "{mod-date}", plain, "cat", now, "2023-07-04"},
		{"mod-weekday", "{mod-weekday}", plain, "cat", now, modTime.Weekday().String()},
		// creation date
		{"created-year", "{created-year}", plain, "cat", now, createdTime.Format("2006")},
		{"created-month", "{created-month}", plain, "cat", now, createdTime.Format("01")},
		{"created-day", "{created-day}", plain, "cat", now, createdTime.Format("02")},
		{"created-date", "{created-date}", plain, "cat", now, createdTime.Format("2006-01-02")},
		// run date
		{"year", "{year}", plain, "cat", now, "2024"},
		{"month", "{month}", plain, "cat", now, "03"},
		{"day", "{day}", plain, "cat", now, "15"},
		{"date", "{date}", plain, "cat", now, "2024-03-15"},
		{"weekday", "{weekday}", plain, "cat", now, "Friday"},
		// run time
		{"hour", "{hour}", plain, "cat", now, "00"},
		{"minute", "{minute}", plain, "cat", now, "00"},
		{"second", "{second}", plain, "cat", now, "00"},
		{"timestamp", "{timestamp}", plain, "cat", now, "20240315-000000"},
		// size
		{"size-range tiny", "{size-range}", tiny, "cat", now, "tiny"},
		{"size-range small", "{size-range}", mid, "cat", now, "small"},
		// category
		{"category", "{category}", plain, "MyCategory", now, "MyCategory"},
		// system context
		{"hostname non-empty", "{hostname}", plain, "cat", now, systemHostname},
		{"username non-empty", "{username}", plain, "cat", now, systemUsername},
		{"os non-empty", "{os}", plain, "cat", now, systemOS},
		// name-trunc
		{"name-trunc:4", "{name-trunc:4}", plain, "docs", now, "my-r"},
		{"name-trunc:100 longer than name", "{name-trunc:100}", plain, "docs", now, "my-report"},
		// combined
		{"combined", "{category}/{year}/{ext}", plain, "docs", now, filepath.FromSlash("docs/2024/pdf")},
		{"combined created path", "{created-year}/{created-month}/{created-day}", plain, "cat", now, filepath.FromSlash(createdTime.Format("2006/01/02"))},
	}
}

// TestResolveGroupBy iterates through the test cases defined in makeResolveGroupByTestCases, sets up the necessary context for each case, and asserts that the output of ResolveGroupBy matches the expected result.
func TestResolveGroupBy(t *testing.T) {
	t.Parallel()
	now := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	modTime := time.Date(2023, 7, 4, 12, 0, 0, 0, time.Local)

	dir := t.TempDir()
	newFile := func(name string, size int, mt time.Time) os.FileInfo {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, make([]byte, size), 0o644))
		if !mt.IsZero() {
			require.NoError(t, os.Chtimes(p, mt, mt))
		}
		info, err := os.Stat(p)
		require.NoError(t, err)
		return info
	}

	plain := newFile("my-report.PDF", 1, modTime)
	tiny := newFile("tiny.bin", 500, modTime)
	mid := newFile("mid.bin", 10*1024*1024, modTime)
	createdTime := getBirthTime(plain)
	initSystemContext()

	for _, tt := range testResolveGroupByTestCases(plain, tiny, mid, createdTime, now, modTime) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveGroupBy(tt.template, &TokenContext{
				Info:         tt.info,
				CategoryName: tt.category,
				Now:          tt.now,
			})
			assert.Equal(t, tt.want, got)
		})
	}
}
