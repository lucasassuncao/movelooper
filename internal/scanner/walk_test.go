package scanner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWalkSource defines the structure for test cases of the WalkSource function,
// containing setup logic, source options resolver, auto-excludes resolver,
// an optional path override, an error expectation flag, and a check function for assertions.
type testWalkSource struct {
	name     string
	setup    func(t *testing.T, root string)
	srcOpts  func(root string) []func(*models.CategorySource)
	excludes func(root string) []string
	path     func(root string) string
	wantErr  bool
	check    func(t *testing.T, entries []scanner.FileEntry, root string)
}

// testWalkSourceTestCases defines a set of test cases for the WalkSource function,
// covering non-recursive, recursive, max depth, auto-exclude, user-defined excludes,
// symlink skipping, empty directory, invalid path, and absolute Dir field scenarios.
var testWalkSourceTestCases = []testWalkSource{
	{
		name: "non-recursive returns only top-level files",
		setup: func(t *testing.T, root string) {
			touch(t, filepath.Join(root, "a.pdf"))
			sub := mkdirAll(t, root, "sub")
			touch(t, filepath.Join(sub, "b.pdf"))
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			require.Len(t, entries, 1)
			assert.Equal(t, "a.pdf", entries[0].Entry.Name())
			assert.Equal(t, root, entries[0].Dir)
		},
	},
	{
		name: "recursive includes subdirectories",
		setup: func(t *testing.T, root string) {
			touch(t, filepath.Join(root, "a.pdf"))
			sub := mkdirAll(t, root, "sub")
			touch(t, filepath.Join(sub, "b.pdf"))
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			names := entryNames(entries)
			assert.Contains(t, names, "a.pdf")
			assert.Contains(t, names, "b.pdf")
		},
	},
	{
		name: "max depth limits descend",
		setup: func(t *testing.T, root string) {
			touch(t, filepath.Join(root, "depth0.txt"))
			sub1 := mkdirAll(t, root, "sub1")
			touch(t, filepath.Join(sub1, "depth1.txt"))
			sub2 := mkdirAll(t, root, "sub1/sub2")
			touch(t, filepath.Join(sub2, "depth2.txt"))
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive, withMaxDepth(1)}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			names := entryNames(entries)
			assert.Contains(t, names, "depth0.txt")
			assert.Contains(t, names, "depth1.txt")
			assert.NotContains(t, names, "depth2.txt")
		},
	},
	{
		name: "auto-exclude skips destination directory",
		setup: func(t *testing.T, root string) {
			touch(t, filepath.Join(root, "a.pdf"))
			dest := mkdirAll(t, root, "Sorted")
			touch(t, filepath.Join(dest, "already-moved.pdf"))
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive}
		},
		excludes: func(root string) []string { return []string{filepath.Join(root, "Sorted")} },
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			names := entryNames(entries)
			assert.Contains(t, names, "a.pdf")
			assert.NotContains(t, names, "already-moved.pdf")
		},
	},
	{
		name: "exclude paths skips user-defined directories",
		setup: func(t *testing.T, root string) {
			touch(t, filepath.Join(root, "a.pdf"))
			archive := mkdirAll(t, root, "Archive")
			touch(t, filepath.Join(archive, "old.pdf"))
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive, withExclude(filepath.Join(root, "Archive"))}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			names := entryNames(entries)
			assert.Contains(t, names, "a.pdf")
			assert.NotContains(t, names, "old.pdf")
		},
	},
	{
		name: "recursive skips symlinks",
		setup: func(t *testing.T, root string) {
			real := filepath.Join(root, "real.pdf")
			touch(t, real)
			link := filepath.Join(root, "link.pdf")
			if err := os.Symlink(real, link); err != nil {
				t.Skip("symlink creation not supported:", err)
			}
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			names := entryNames(entries)
			assert.Contains(t, names, "real.pdf")
			assert.NotContains(t, names, "link.pdf")
		},
	},
	{
		name:  "empty dir returns empty",
		setup: func(t *testing.T, root string) {},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			assert.Empty(t, entries)
		},
	},
	{
		name:    "invalid path returns error",
		setup:   func(t *testing.T, root string) {},
		path:    func(root string) string { return "/nonexistent/path/abc123" },
		wantErr: true,
	},
	{
		name: "file entry dir is absolute",
		setup: func(t *testing.T, root string) {
			sub := mkdirAll(t, root, "sub")
			touch(t, filepath.Join(sub, "file.txt"))
		},
		srcOpts: func(root string) []func(*models.CategorySource) {
			return []func(*models.CategorySource){withRecursive}
		},
		check: func(t *testing.T, entries []scanner.FileEntry, root string) {
			require.Len(t, entries, 1)
			assert.True(t, filepath.IsAbs(entries[0].Dir))
			assert.Equal(t, filepath.Join(root, "sub"), entries[0].Dir)
		},
	},
}

// TestWalkSource tests the WalkSource function with various source configurations
// to ensure it correctly walks directories and applies filters.
func TestWalkSource(t *testing.T) {
	t.Parallel()
	for _, tt := range testWalkSourceTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			tt.setup(t, root)

			path := root
			if tt.path != nil {
				path = tt.path(root)
			}

			var opts []func(*models.CategorySource)
			if tt.srcOpts != nil {
				opts = tt.srcOpts(root)
			}

			var excludes []string
			if tt.excludes != nil {
				excludes = tt.excludes(root)
			}

			entries, err := scanner.WalkSource(context.Background(), src(path, opts...), excludes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, entries, root)
			}
		})
	}
}

// Helper functions for test setup and assertions

// withExclude returns a function that sets the ExcludePaths field of a CategorySource to the specified paths.
func withExclude(paths ...string) func(*models.CategorySource) {
	return func(s *models.CategorySource) {
		s.ExcludePaths = paths
	}
}

// withMaxDepth returns a function that sets the MaxDepth field of a CategorySource to the specified value.
func withMaxDepth(n int) func(*models.CategorySource) {
	return func(s *models.CategorySource) {
		s.MaxDepth = n
	}
}

// withRecursive sets the Recursive field of a CategorySource to true, enabling recursive directory walking.
func withRecursive(s *models.CategorySource) {
	s.Recursive = true
}

// entryNames extracts the names of the entries from a slice of FileEntry and returns them as a slice of strings.
func entryNames(entries []scanner.FileEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Entry.Name()
	}
	return names
}

// touch creates an empty file at the specified path, ensuring the parent directory exists.
func mkdirAll(t *testing.T, base, rel string) string {
	t.Helper()
	p := filepath.Join(base, rel)
	require.NoError(t, os.MkdirAll(p, 0o755))
	return p
}

// touch creates an empty file at the specified path, ensuring the parent directory exists.
func src(path string, opts ...func(*models.CategorySource)) models.CategorySource {
	s := models.CategorySource{Path: path}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// touch creates an empty file at the given path.
func touch(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
}
