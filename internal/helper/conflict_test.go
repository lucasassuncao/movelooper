package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, content, 0644))
}

// --- getUniqueDestinationPath ---

func TestGetUniqueDestinationPath(t *testing.T) {
	tests := []struct {
		name     string
		existing []string // files to pre-create
		input    string
		want     string // expected basename
	}{
		{"no conflict", nil, "file.txt", "file.txt"},
		{"one conflict", []string{"file.txt"}, "file.txt", "file(1).txt"},
		{"multiple conflicts", []string{"file.txt", "file(1).txt", "file(2).txt"}, "file.txt", "file(3).txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// --- resolveConflict dispatch ---

func TestResolveConflict(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		wantMove   bool
		wantSuffix string
	}{
		{"unknown falls to rename", "unknown_strategy", true, "dst(1).txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src.txt")
			dst := filepath.Join(dir, "dst.txt")
			writeFile(t, src, []byte("src"))
			writeFile(t, dst, []byte("dst"))

			path, shouldMove, err := resolveConflict(tt.strategy, src, dst, dir, "dst.txt")
			require.NoError(t, err)
			assert.Equal(t, tt.wantMove, shouldMove)
			assert.Contains(t, path, tt.wantSuffix)
		})
	}
}

// --- individual resolvers ---

func TestResolvers(t *testing.T) {
	type want struct {
		move       bool
		pathIsDst  bool   // result == dst
		pathSuffix string // substring in result (when not dst and not empty)
		srcRemoved bool
		dstRemoved bool
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T, src, dst string)
		resolve func(src, dst, dir, name string) (string, bool, error)
		want    want
	}{
		{
			name: "rename/conflict renames",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("existing"))
			},
			resolve: (&renameResolver{}).Resolve,
			want:    want{move: true, pathSuffix: "(1)"},
		},
		{
			name: "overwrite/removes dst",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("old"))
			},
			resolve: (&overwriteResolver{}).Resolve,
			want:    want{move: true, pathIsDst: true, dstRemoved: true},
		},
		{
			name: "skip/does not move",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("dst"))
			},
			resolve: (&skipResolver{}).Resolve,
			want:    want{move: false},
		},
		{
			name: "hash_check/duplicate removes src",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("identical"))
				writeFile(t, dst, []byte("identical"))
			},
			resolve: (&hashCheckResolver{}).Resolve,
			want:    want{move: false, srcRemoved: true},
		},
		{
			name: "hash_check/different renames",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("content A"))
				writeFile(t, dst, []byte("content B"))
			},
			resolve: (&hashCheckResolver{}).Resolve,
			want:    want{move: true, pathSuffix: "(1)"},
		},
		{
			name: "newest/src newer moves",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("dst"))
				require.NoError(t, os.Chtimes(dst, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			resolve: (&newestResolver{}).Resolve,
			want:    want{move: true, pathIsDst: true},
		},
		{
			name: "newest/dst newer skips",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("dst"))
				require.NoError(t, os.Chtimes(src, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			resolve: (&newestResolver{}).Resolve,
			want:    want{move: false},
		},
		{
			name: "oldest/src older moves",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("dst"))
				require.NoError(t, os.Chtimes(src, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			resolve: (&oldestResolver{}).Resolve,
			want:    want{move: true, pathIsDst: true},
		},
		{
			name: "oldest/dst older skips",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, []byte("src"))
				writeFile(t, dst, []byte("dst"))
				require.NoError(t, os.Chtimes(dst, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			resolve: (&oldestResolver{}).Resolve,
			want:    want{move: false},
		},
		{
			name: "larger/src larger moves",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, make([]byte, 200))
				writeFile(t, dst, make([]byte, 100))
			},
			resolve: (&largerResolver{}).Resolve,
			want:    want{move: true, pathIsDst: true},
		},
		{
			name: "larger/dst larger skips",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, make([]byte, 100))
				writeFile(t, dst, make([]byte, 200))
			},
			resolve: (&largerResolver{}).Resolve,
			want:    want{move: false},
		},
		{
			name: "smaller/src smaller moves",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, make([]byte, 100))
				writeFile(t, dst, make([]byte, 200))
			},
			resolve: (&smallerResolver{}).Resolve,
			want:    want{move: true, pathIsDst: true},
		},
		{
			name: "smaller/dst smaller skips",
			setup: func(t *testing.T, src, dst string) {
				writeFile(t, src, make([]byte, 200))
				writeFile(t, dst, make([]byte, 100))
			},
			resolve: (&smallerResolver{}).Resolve,
			want:    want{move: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src.txt")
			dst := filepath.Join(dir, "dst.txt")
			tt.setup(t, src, dst)

			path, shouldMove, err := tt.resolve(src, dst, dir, "dst.txt")
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
