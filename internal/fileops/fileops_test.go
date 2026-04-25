package fileops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *pterm.Logger {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &l
}

func newTestMoveContext() MoveContext {
	return MoveContext{Logger: newTestLogger()}
}

// --- CreateDirectory ---

func TestCreateDirectory(t *testing.T) {
	tests := []struct {
		name    string
		path    func(base string) string
		wantErr bool
	}{
		{"creates nested dir", func(base string) string { return filepath.Join(base, "sub", "dir") }, false},
		{"idempotent on existing", func(base string) string { return base }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.path(t.TempDir())
			err := CreateDirectory(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			info, err := os.Stat(dir)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

// --- ReadDirectory ---

func TestReadDirectory(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantLen     int
		wantErr     bool
		nonExistent bool
	}{
		{
			name: "returns entries",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644))
			},
			wantLen: 2,
		},
		{
			name:        "non-existent returns error",
			nonExistent: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.nonExistent {
				dir = filepath.Join(dir, "nonexistent")
			} else if tt.setup != nil {
				tt.setup(t, dir)
			}
			entries, err := ReadDirectory(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, entries, tt.wantLen)
		})
	}
}

// --- copyFile ---

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		check   func(t *testing.T, src, dst string)
	}{
		{
			name:    "copies content",
			content: []byte("hello world"),
			check: func(t *testing.T, _, dst string) {
				got, err := os.ReadFile(dst)
				require.NoError(t, err)
				assert.Equal(t, []byte("hello world"), got)
			},
		},
		{
			name:    "preserves mod time",
			content: []byte("data"),
			check: func(t *testing.T, src, dst string) {
				srcInfo, err := os.Stat(src)
				require.NoError(t, err)
				dstInfo, err := os.Stat(dst)
				require.NoError(t, err)
				assert.Equal(t, srcInfo.ModTime().Unix(), dstInfo.ModTime().Unix())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src.txt")
			dst := filepath.Join(dir, "dst.txt")
			require.NoError(t, os.WriteFile(src, tt.content, 0644))
			require.NoError(t, copyFile(context.Background(), src, dst))
			tt.check(t, src, dst)
		})
	}
}

// --- MoveFiles ---

func TestMoveFiles(t *testing.T) {
	enabled := true

	tests := []struct {
		name      string
		setup     func(t *testing.T, src, dst string)
		category  func(src, dst string) *models.Category
		ext       string
		batchID   string
		wantMoved []string
		check     func(t *testing.T, src, dst string)
	}{
		{
			name: "moves matching extension",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "img.jpg"), []byte("jpg"), 0644))
			},
			category: func(src, dst string) *models.Category {
				return &models.Category{
					Name: "PDFs", Enabled: &enabled,
					Source:      models.CategorySource{Path: src, Extensions: []string{"pdf"}},
					Destination: models.CategoryDestination{Path: dst, ConflictStrategy: "rename"},
				}
			},
			ext: "pdf", batchID: "batch_test",
			wantMoved: []string{"doc.pdf"},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "doc.pdf"))
				assert.NoFileExists(t, filepath.Join(src, "doc.pdf"))
				assert.FileExists(t, filepath.Join(src, "img.jpg"))
			},
		},
		{
			name: "skip strategy leaves src",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("existing"), 0644))
			},
			category: func(src, dst string) *models.Category {
				return &models.Category{
					Name: "Texts", Enabled: &enabled,
					Source:      models.CategorySource{Path: src, Extensions: []string{"txt"}},
					Destination: models.CategoryDestination{Path: dst, ConflictStrategy: "skip"},
				}
			},
			ext: "txt", batchID: "batch_skip",
			wantMoved: nil,
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(src, "file.txt"))
			},
		},
		{
			name: "organize-by places in subdir",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("img"), 0644))
			},
			category: func(src, dst string) *models.Category {
				return &models.Category{
					Name: "Photos", Enabled: &enabled,
					Source:      models.CategorySource{Path: src, Extensions: []string{"jpg"}},
					Destination: models.CategoryDestination{Path: dst, OrganizeBy: "{ext}", ConflictStrategy: "rename"},
				}
			},
			ext: "jpg", batchID: "batch_org",
			wantMoved: []string{"photo.jpg"},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "jpg", "photo.jpg"))
			},
		},
		{
			name: "ext all moves every file",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "b.pdf"), []byte("b"), 0644))
			},
			category: func(src, dst string) *models.Category {
				return &models.Category{
					Name: "All", Enabled: &enabled,
					Source:      models.CategorySource{Path: src},
					Destination: models.CategoryDestination{Path: dst, ConflictStrategy: "rename"},
				}
			},
			ext: "all", batchID: "batch_all",
			check: func(t *testing.T, src, dst string) {
				assert.Len(t, func() []string {
					entries, _ := filepath.Glob(filepath.Join(dst, "*"))
					return entries
				}(), 2)
			},
		},
		{
			name: "empty strategy defaults to rename",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("existing"), 0644))
			},
			category: func(src, dst string) *models.Category {
				return &models.Category{
					Name: "Texts", Enabled: &enabled,
					Source:      models.CategorySource{Path: src, Extensions: []string{"txt"}},
					Destination: models.CategoryDestination{Path: dst, ConflictStrategy: ""},
				}
			},
			ext: "txt", batchID: "batch_default",
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "file(1).txt"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()
			tt.setup(t, src, dst)

			entries, err := os.ReadDir(src)
			require.NoError(t, err)

			cat := tt.category(src, dst)
			req := MoveRequest{
				Category:  cat,
				Files:     entries,
				Extension: tt.ext,
				BatchID:   tt.batchID,
				SourceDir: src,
			}
			result := MoveFiles(context.Background(), newTestMoveContext(), req)

			if tt.wantMoved != nil {
				assert.Equal(t, tt.wantMoved, result.Moved)
			}
			if tt.check != nil {
				tt.check(t, src, dst)
			}
		})
	}
}

// --- applyConflictStrategy ---

func TestApplyConflictStrategy(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		setup      func(t *testing.T, srcFile, dstFile string)
		wantSkip   bool
		wantEqDst  bool
		wantSuffix string
	}{
		{
			name:      "no conflict returns dst as-is",
			strategy:  "rename",
			setup:     func(t *testing.T, srcFile, dstFile string) { writeFile(t, srcFile, []byte("data")) },
			wantEqDst: true,
		},
		{
			name:     "skip strategy skips",
			strategy: "skip",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("new"))
				writeFile(t, dstFile, []byte("existing"))
			},
			wantSkip: true,
		},
		{
			name:     "rename strategy renames",
			strategy: "rename",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("new"))
				writeFile(t, dstFile, []byte("existing"))
			},
			wantSuffix: "(1)",
		},
		{
			name:     "overwrite strategy returns dst",
			strategy: "overwrite",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("new"))
				writeFile(t, dstFile, []byte("existing"))
			},
			wantEqDst: true,
		},
		{
			name:     "hash_check duplicate skips and removes src",
			strategy: "hash_check",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("identical content"))
				writeFile(t, dstFile, []byte("identical content"))
			},
			wantSkip: true,
		},
		{
			name:     "newest/src newer moves to dst",
			strategy: "newest",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("new"))
				writeFile(t, dstFile, []byte("old"))
				require.NoError(t, os.Chtimes(dstFile, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			wantEqDst: true,
		},
		{
			name:     "oldest/src older moves to dst",
			strategy: "oldest",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("old"))
				writeFile(t, dstFile, []byte("new"))
				require.NoError(t, os.Chtimes(srcFile, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
			},
			wantEqDst: true,
		},
		{
			name:     "larger/src larger moves to dst",
			strategy: "larger",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("larger content here"))
				writeFile(t, dstFile, []byte("small"))
			},
			wantEqDst: true,
		},
		{
			name:     "smaller/src smaller moves to dst",
			strategy: "smaller",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("tiny"))
				writeFile(t, dstFile, []byte("much larger content here"))
			},
			wantEqDst: true,
		},
		{
			name:     "unknown strategy falls to rename",
			strategy: "does_not_exist",
			setup: func(t *testing.T, srcFile, dstFile string) {
				writeFile(t, srcFile, []byte("x"))
				writeFile(t, dstFile, []byte("y"))
			},
			wantSuffix: "(1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()
			srcFile := filepath.Join(src, "file.txt")
			dstFile := filepath.Join(dst, "file.txt")
			tt.setup(t, srcFile, dstFile)

			resolved, skip := applyConflictStrategy(newTestMoveContext(), tt.strategy, ConflictArgs{
				Src: srcFile, Dst: dstFile, DestDir: dst, FileName: "file.txt",
			})
			assert.Equal(t, tt.wantSkip, skip)
			if !tt.wantSkip {
				switch {
				case tt.wantEqDst:
					assert.Equal(t, dstFile, resolved)
				case tt.wantSuffix != "":
					assert.Contains(t, resolved, tt.wantSuffix)
				}
			}
		})
	}
}

// --- isCrossDeviceError ---

func TestIsCrossDeviceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"non-link error", os.ErrPermission, false},
		{"link error with permission", &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrPermission}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCrossDeviceError(tt.err))
		})
	}
}

func TestDispatchAction_Move(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	dst := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	require.NoError(t, dispatchAction(context.Background(), "move", src, dst))

	assert.FileExists(t, dst)
	assert.NoFileExists(t, src)
}

func TestDispatchAction_Copy(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	dst := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	require.NoError(t, dispatchAction(context.Background(), "copy", src, dst))

	assert.FileExists(t, dst)
	assert.FileExists(t, src)
	srcData, _ := os.ReadFile(src)
	dstData, _ := os.ReadFile(dst)
	assert.Equal(t, srcData, dstData)
}

func TestDispatchAction_Symlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "file.txt")
	dst := filepath.Join(dir, "link.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	err := dispatchAction(context.Background(), "symlink", src, dst)
	if err != nil {
		t.Skipf("symlink not available (likely missing privilege on Windows): %v", err)
	}

	info, err := os.Lstat(dst)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "dst should be a symlink")
}

func TestFileActions_UnknownDefaultsToMove(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	dst := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))

	require.NoError(t, dispatchAction(context.Background(), "unknown_action", src, dst))

	assert.FileExists(t, dst)
	assert.NoFileExists(t, src)
}

func TestFileActions_EmptyDefaultsToMove(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.txt")
	dst := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))

	require.NoError(t, dispatchAction(context.Background(), "", src, dst))

	assert.FileExists(t, dst)
	assert.NoFileExists(t, src)
}

func readDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	return entries
}

func TestMoveFiles_RenameTemplate(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("x"), 0644))

	cat := &models.Category{
		Name: "images",
		Source: models.CategorySource{
			Path:       src,
			Extensions: []string{"jpg"},
		},
		Destination: models.CategoryDestination{
			Path:   dst,
			Action: "copy",
			Rename: "{category}_{name}.{ext}",
		},
	}

	req := MoveRequest{
		Category:  cat,
		Files:     readDir(t, src),
		Extension: "jpg",
		BatchID:   "batch_test",
		SourceDir: src,
	}
	result := MoveFiles(context.Background(), newTestMoveContext(), req)

	require.Len(t, result.Moved, 1)
	assert.FileExists(t, filepath.Join(dst, "images_photo.jpg"))
	assert.FileExists(t, filepath.Join(src, "photo.jpg"))
}
