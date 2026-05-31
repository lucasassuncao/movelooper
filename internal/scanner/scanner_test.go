package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testScan defines the structure for test cases of the Scan function,
// containing setup logic, an optional path resolver (defaults to the temp dir),
// an error expectation flag, and a check function for assertions.
type testScan struct {
	name    string
	setup   func(t *testing.T, dir string)
	path    func(dir string) string
	wantErr bool
	check   func(t *testing.T, result scanner.Result)
}

// testScanTestCases defines a set of test cases for the Scan function,
// covering extension detection, case insensitivity, empty dirs, unknown extensions,
// subdirectory skipping, dictionary order, and error scenarios.
var testScanTestCases = []testScan{
	{
		name: "detects known extensions",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "photo.jpg"))
			touch(t, filepath.Join(dir, "clip.mp4"))
			touch(t, filepath.Join(dir, "song.mp3"))
		},
		check: func(t *testing.T, result scanner.Result) {
			names := categoryNames(result)
			assert.Contains(t, names, "images")
			assert.Contains(t, names, "videos")
			assert.Contains(t, names, "audio")
		},
	},
	{
		name: "case insensitive extension",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "PHOTO.JPG"))
			touch(t, filepath.Join(dir, "doc.PDF"))
		},
		check: func(t *testing.T, result scanner.Result) {
			names := categoryNames(result)
			assert.Contains(t, names, "images")
			assert.Contains(t, names, "documents")
		},
	},
	{
		name:  "empty dir returns no categories",
		setup: func(t *testing.T, dir string) {},
		check: func(t *testing.T, result scanner.Result) {
			assert.Empty(t, result.Categories)
		},
	},
	{
		name: "unknown extension only returns no categories",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "data.xyz123"))
		},
		check: func(t *testing.T, result scanner.Result) {
			assert.Empty(t, result.Categories)
		},
	},
	{
		name: "skips subdirectories",
		setup: func(t *testing.T, dir string) {
			subdir := filepath.Join(dir, "subdir")
			require.NoError(t, os.Mkdir(subdir, 0755))
			touch(t, filepath.Join(subdir, "nested.jpg"))
		},
		check: func(t *testing.T, result scanner.Result) {
			assert.Empty(t, result.Categories)
		},
	},
	{
		name: "preserves dictionary order",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "a.mp3"))
			touch(t, filepath.Join(dir, "b.jpg"))
			touch(t, filepath.Join(dir, "c.epub"))
		},
		check: func(t *testing.T, result scanner.Result) {
			require.Len(t, result.Categories, 3)
			assert.Equal(t, "images", result.Categories[0].Name)
			assert.Equal(t, "audio", result.Categories[1].Name)
			assert.Equal(t, "ebooks", result.Categories[2].Name)
		},
	},
	{
		name: "only found extensions returned",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "photo.jpg"))
		},
		check: func(t *testing.T, result scanner.Result) {
			require.Len(t, result.Categories, 1)
			assert.Equal(t, []string{"jpg"}, result.Categories[0].Extensions)
		},
	},
	{
		name:    "path does not exist returns error",
		setup:   func(t *testing.T, dir string) {},
		path:    func(dir string) string { return "/this/path/does/not/exist/ever" },
		wantErr: true,
	},
	{
		name: "path is file returns error",
		setup: func(t *testing.T, dir string) {
			touch(t, filepath.Join(dir, "file.txt"))
		},
		path:    func(dir string) string { return filepath.Join(dir, "file.txt") },
		wantErr: true,
	},
}

// TestScan tests the Scan function with various directory configurations
// to ensure it correctly detects file categories.
func TestScan(t *testing.T) {
	for _, tt := range testScanTestCases {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)

			path := dir
			if tt.path != nil {
				path = tt.path(dir)
			}

			result, err := scanner.Scan(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// Helper functions for TestWalkSource

// categoryNames is a helper function that extracts the category names from a scanner.Result for easier assertions in tests.
func categoryNames(result scanner.Result) []string {
	names := make([]string, len(result.Categories))
	for i, c := range result.Categories {
		names[i] = c.Name
	}
	return names
}

// touch creates an empty file at path.
func touch(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))
}
