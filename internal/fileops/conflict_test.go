package fileops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConflictResolverSkipMessage defines the structure for test cases of the SkipMessage method,
// containing the resolver under test and the expected message.
type testConflictResolverSkipMessage struct {
	name     string
	resolver ConflictResolver
	wantMsg  string
}

// testConflictResolverSkipMessageTestCases defines a set of test cases for the SkipMessage method
// across all resolver types.
var testConflictResolverSkipMessageTestCases = []testConflictResolverSkipMessage{
	{"skip", &skipResolver{}, "file skipped due to conflict strategy"},
	{"newest", newestResolver, "file skipped - destination is newer"},
	{"oldest", oldestResolver, "file skipped - destination is older"},
	{"larger", largerResolver, "file skipped - destination is larger"},
	{"smaller", smallerResolver, "file skipped - destination is smaller"},
	{"hash_check", &hashCheckResolver{}, "duplicate file removed from source"},
	{"rename", &renameResolver{}, ""},
	{"overwrite", &overwriteResolver{}, ""},
}

// TestConflictResolvers_SkipMessages tests the SkipMessage method for all conflict resolvers
// to ensure each returns the correct message.
func TestConflictResolvers_SkipMessages(t *testing.T) {
	t.Parallel()
	for _, tt := range testConflictResolverSkipMessageTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantMsg, tt.resolver.SkipMessage())
		})
	}
}

// testGetUniqueDestinationPath defines the structure for test cases of the getUniqueDestinationPath function,
// containing existing files, input filename, and expected output filename.
type testGetUniqueDestinationPath struct {
	name     string
	existing []string
	input    string
	want     string
}

// testGetUniqueDestinationPathTestCases defines a set of test cases for the getUniqueDestinationPath function,
// covering no conflict, one conflict, and multiple sequential conflicts.
var testGetUniqueDestinationPathTestCases = []testGetUniqueDestinationPath{
	{"no conflict", nil, "file.txt", "file.txt"},
	{"one conflict", []string{"file.txt"}, "file.txt", "file(1).txt"},
	{"multiple conflicts", []string{"file.txt", "file(1).txt", "file(2).txt"}, "file.txt", "file(3).txt"},
}

// TestGetUniqueDestinationPath tests the getUniqueDestinationPath function to ensure it correctly
// generates unique filenames when conflicts exist.
func TestGetUniqueDestinationPath(t *testing.T) {
	t.Parallel()
	for _, tt := range testGetUniqueDestinationPathTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.existing {
				writeFile(t, filepath.Join(dir, f), []byte("x"))
			}
			got, err := getUniqueDestinationPath(dir, tt.input)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(dir, tt.want), got)
		})
	}
}

// testResolver defines the structure for test cases of individual conflict resolvers,
// containing setup logic, the resolve function, and expected outcome fields.
type testResolver struct {
	name    string
	setup   func(t *testing.T, src, dst string)
	resolve func(ConflictArgs) (string, bool, FinalizeFunc, error)
	want    testResolverWant
}

// testResolverWant defines the expected outcome fields for resolver test cases.
type testResolverWant struct {
	move       bool
	pathIsDst  bool
	pathSuffix string
	srcRemoved bool
	dstRemoved bool
}

// testResolverTestCases defines a set of test cases for all conflict resolver implementations,
// covering rename, overwrite, skip, hash_check, newest, oldest, larger, and smaller strategies.
var testResolverTestCases = []testResolver{
	{
		name: "rename/conflict renames",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("existing"))
		},
		resolve: (&renameResolver{}).Resolve,
		want:    testResolverWant{move: true, pathSuffix: "(1)"},
	},
	{
		name: "overwrite/resolves to dst",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("old"))
		},
		resolve: (&overwriteResolver{}).Resolve,
		// On POSIX os.Rename replaces atomically (dst left in place); on Windows
		// Resolve moves dst aside under a backup. Either way resolved == dst.
		want: testResolverWant{move: true, pathIsDst: true},
	},
	{
		name: "skip/does not move",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))
		},
		resolve: (&skipResolver{}).Resolve,
		want:    testResolverWant{move: false},
	},
	{
		name: "hash_check/duplicate removes src",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("identical"))
			writeFile(t, dst, []byte("identical"))
		},
		resolve: (&hashCheckResolver{}).Resolve,
		want:    testResolverWant{move: false, srcRemoved: true},
	},
	{
		name: "hash_check/different renames",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("content A"))
			writeFile(t, dst, []byte("content B"))
		},
		resolve: (&hashCheckResolver{}).Resolve,
		want:    testResolverWant{move: true, pathSuffix: "(1)"},
	},
	{
		name: "newest/src newer moves",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))
			require.NoError(t, os.Chtimes(dst, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
		},
		resolve: (newestResolver).Resolve,
		want:    testResolverWant{move: true, pathIsDst: true},
	},
	{
		name: "newest/dst newer skips",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))
			require.NoError(t, os.Chtimes(src, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
		},
		resolve: (newestResolver).Resolve,
		want:    testResolverWant{move: false},
	},
	{
		name: "oldest/src older moves",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))
			require.NoError(t, os.Chtimes(src, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
		},
		resolve: (oldestResolver).Resolve,
		want:    testResolverWant{move: true, pathIsDst: true},
	},
	{
		name: "oldest/dst older skips",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))
			require.NoError(t, os.Chtimes(dst, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
		},
		resolve: (oldestResolver).Resolve,
		want:    testResolverWant{move: false},
	},
	{
		name: "larger/src larger moves",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, make([]byte, 200))
			writeFile(t, dst, make([]byte, 100))
		},
		resolve: (largerResolver).Resolve,
		want:    testResolverWant{move: true, pathIsDst: true},
	},
	{
		name: "larger/dst larger skips",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, make([]byte, 100))
			writeFile(t, dst, make([]byte, 200))
		},
		resolve: (largerResolver).Resolve,
		want:    testResolverWant{move: false},
	},
	{
		name: "smaller/src smaller moves",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, make([]byte, 100))
			writeFile(t, dst, make([]byte, 200))
		},
		resolve: (smallerResolver).Resolve,
		want:    testResolverWant{move: true, pathIsDst: true},
	},
	{
		name: "smaller/dst smaller skips",
		setup: func(t *testing.T, src, dst string) {
			writeFile(t, src, make([]byte, 200))
			writeFile(t, dst, make([]byte, 100))
		},
		resolve: (smallerResolver).Resolve,
		want:    testResolverWant{move: false},
	},
}

// TestResolvers tests all conflict resolver implementations to ensure they correctly
// handle move, skip, rename, and removal scenarios.
func TestResolvers(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolverTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			src := filepath.Join(dir, "src.txt")
			dst := filepath.Join(dir, "dst.txt")
			tt.setup(t, src, dst)

			path, shouldMove, _, err := tt.resolve(ConflictArgs{Src: src, Dst: dst, DestDir: dir, FileName: "dst.txt"})
			require.NoError(t, err)
			assert.Equal(t, tt.want.move, shouldMove)

			switch {
			case tt.want.pathIsDst:
				assert.Equal(t, dst, path)
			case tt.want.pathSuffix != "":
				assert.Contains(t, path, tt.want.pathSuffix)
			default:
				assert.Empty(t, path)
			}

			if tt.want.srcRemoved {
				assert.NoFileExists(t, src)
			}
			if tt.want.dstRemoved {
				assert.NoFileExists(t, dst)
			}
		})
	}
}

// TestSafeSwap verifies that a replace-style resolver moves the existing
// destination aside and that the returned finalize either restores it (when the
// action fails) or discards it (when the action succeeds), so a failed action
// never destroys the previous destination file.
func TestSafeSwap(t *testing.T) {
	t.Parallel()

	t.Run("restores original destination when action fails", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, make([]byte, 200)) // src larger → larger strategy moves
		writeFile(t, dst, []byte("original"))

		_, shouldMove, finalize, err := (largerResolver).Resolve(
			ConflictArgs{Src: src, Dst: dst, DestDir: dir, FileName: "dst.txt"})
		require.NoError(t, err)
		require.True(t, shouldMove)
		require.NotNil(t, finalize)

		// resolver moved dst aside, freeing the original path for the action.
		assert.NoFileExists(t, dst)

		// the action failed → finalize must restore the original destination.
		require.NoError(t, finalize(true))
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, []byte("original"), got)
	})

	t.Run("discards set-aside destination when action succeeds", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		writeFile(t, src, make([]byte, 200))
		writeFile(t, dst, []byte("original"))

		_, _, finalize, err := (largerResolver).Resolve(
			ConflictArgs{Src: src, Dst: dst, DestDir: dir, FileName: "dst.txt"})
		require.NoError(t, err)
		require.NotNil(t, finalize)

		// simulate a successful action writing a fresh destination.
		writeFile(t, dst, []byte("new content"))
		require.NoError(t, finalize(false))

		// the backup is gone and only the new destination remains.
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			assert.NotContains(t, e.Name(), ".ml-bak", "backup should be removed on success")
		}
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, []byte("new content"), got)
	})
}

// TestCompareFileHashes covers content equality, including the size-mismatch
// short-circuit that returns "not equal" without hashing either file.
func TestCompareFileHashes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b []byte
		want bool
	}{
		{"identical content", []byte("same bytes"), []byte("same bytes"), true},
		{"same size, different content", []byte("content A"), []byte("content B"), false},
		{"different sizes short-circuit", []byte("short"), []byte("a much longer body"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			a := filepath.Join(dir, "a.bin")
			b := filepath.Join(dir, "b.bin")
			writeFile(t, a, tc.a)
			writeFile(t, b, tc.b)

			got, err := compareFileHashes(a, b)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, content, 0o644))
}
