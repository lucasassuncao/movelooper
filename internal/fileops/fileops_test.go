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

// testCreateDirectory defines the structure for test cases of the CreateDirectory function,
// containing a path builder and an error expectation flag.
type testCreateDirectory struct {
	name    string
	path    func(base string) string
	wantErr bool
}

// testCreateDirectoryTestCases defines a set of test cases for the CreateDirectory function,
// covering nested directory creation and idempotent behavior on existing directories.
var testCreateDirectoryTestCases = []testCreateDirectory{
	{"creates nested dir", func(base string) string { return filepath.Join(base, "sub", "dir") }, false},
	{"idempotent on existing", func(base string) string { return base }, false},
}

// TestCreateDirectory tests the CreateDirectory function to ensure it correctly creates directories.
func TestCreateDirectory(t *testing.T) {
	for _, tt := range testCreateDirectoryTestCases {
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

// testReadDirectory defines the structure for test cases of the ReadDirectory function,
// containing setup logic, expected entry count, error expectation, and a non-existent path flag.
type testReadDirectory struct {
	name        string
	setup       func(t *testing.T, dir string)
	wantLen     int
	wantErr     bool
	nonExistent bool
}

// testReadDirectoryTestCases defines a set of test cases for the ReadDirectory function,
// covering populated directory and non-existent path scenarios.
var testReadDirectoryTestCases = []testReadDirectory{
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

// TestReadDirectory tests the ReadDirectory function to ensure it correctly reads directory entries.
func TestReadDirectory(t *testing.T) {
	for _, tt := range testReadDirectoryTestCases {
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

// testCopyFile defines the structure for test cases of the copyFile function,
// containing file content and a check function for assertions on the copied file.
type testCopyFile struct {
	name    string
	content []byte
	check   func(t *testing.T, src, dst string)
}

// testCopyFileTestCases defines a set of test cases for the copyFile function,
// covering content correctness and modification time preservation.
var testCopyFileTestCases = []testCopyFile{
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

// TestCopyFile tests the copyFile function to ensure it correctly copies file content and preserves metadata.
func TestCopyFile(t *testing.T) {
	for _, tt := range testCopyFileTestCases {
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

// testMoveFiles defines the structure for test cases of the MoveFiles function,
// containing setup logic, a category builder, extension, batch ID, expected moved files, and a check function.
type testMoveFiles struct {
	name      string
	setup     func(t *testing.T, src, dst string)
	category  func(src, dst string) *models.Category
	ext       string
	batchID   string
	wantMoved []string
	check     func(t *testing.T, src, dst string)
}

// testMoveFilesTestCases defines a set of test cases for the MoveFiles function,
// covering extension matching, skip strategy, organize-by, all extension, and empty conflict strategy.
var testMoveFilesTestCases = []testMoveFiles{
	{
		name: "moves matching extension",
		setup: func(t *testing.T, src, dst string) {
			require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(src, "img.jpg"), []byte("jpg"), 0644))
		},
		category: func(src, dst string) *models.Category {
			enabled := true
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
			enabled := true
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
			enabled := true
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
			enabled := true
			return &models.Category{
				Name: "All", Enabled: &enabled,
				Source:      models.CategorySource{Path: src},
				Destination: models.CategoryDestination{Path: dst, ConflictStrategy: "rename"},
			}
		},
		ext: "all", batchID: "batch_all",
		check: func(t *testing.T, src, dst string) {
			entries, _ := filepath.Glob(filepath.Join(dst, "*"))
			assert.Len(t, entries, 2)
		},
	},
	{
		name: "empty strategy defaults to rename",
		setup: func(t *testing.T, src, dst string) {
			require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
			require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("existing"), 0644))
		},
		category: func(src, dst string) *models.Category {
			enabled := true
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
	{
		name: "rename template generates correct filename",
		setup: func(t *testing.T, src, dst string) {
			require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("x"), 0644))
		},
		category: func(src, dst string) *models.Category {
			return &models.Category{
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
		},
		ext: "jpg", batchID: "batch_rename",
		check: func(t *testing.T, src, dst string) {
			assert.FileExists(t, filepath.Join(dst, "images_photo.jpg"))
			assert.FileExists(t, filepath.Join(src, "photo.jpg"))
		},
	},
}

// TestMoveFiles tests the MoveFiles function with various category configurations
// to ensure it correctly moves files and applies conflict strategies.
func TestMoveFiles(t *testing.T) {
	for _, tt := range testMoveFilesTestCases {
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

// testApplyConflictStrategy defines the structure for test cases of the applyConflictStrategy function,
// containing the strategy, setup logic, and expected outcome fields.
type testApplyConflictStrategy struct {
	name       string
	strategy   string
	setup      func(t *testing.T, srcFile, dstFile string)
	wantSkip   bool
	wantEqDst  bool
	wantSuffix string
}

// testApplyConflictStrategyTestCases defines a set of test cases for the applyConflictStrategy function,
// covering no conflict, skip, rename, overwrite, hash_check, newest, oldest, larger, smaller, and unknown strategy.
var testApplyConflictStrategyTestCases = []testApplyConflictStrategy{
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
			oneHourAgo := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(dstFile, oneHourAgo, oneHourAgo))
		},
		wantEqDst: true,
	},
	{
		name:     "oldest/src older moves to dst",
		strategy: "oldest",
		setup: func(t *testing.T, srcFile, dstFile string) {
			writeFile(t, srcFile, []byte("old"))
			writeFile(t, dstFile, []byte("new"))
			oneHourAgo := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(srcFile, oneHourAgo, oneHourAgo))
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

// TestApplyConflictStrategy tests the applyConflictStrategy function with all strategy types
// to ensure it correctly resolves conflicts and returns the appropriate destination path.
func TestApplyConflictStrategy(t *testing.T) {
	for _, tt := range testApplyConflictStrategyTestCases {
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

// testIsCrossDeviceError defines the structure for test cases of the isCrossDeviceError function,
// containing the error and expected result.
type testIsCrossDeviceError struct {
	name string
	err  error
	want bool
}

// testIsCrossDeviceErrorTestCases defines a set of test cases for the isCrossDeviceError function,
// covering nil, non-link errors, and link errors.
var testIsCrossDeviceErrorTestCases = []testIsCrossDeviceError{
	{"nil", nil, false},
	{"non-link error", os.ErrPermission, false},
	{"link error with permission", &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrPermission}, false},
}

// TestIsCrossDeviceError tests the isCrossDeviceError function to ensure it correctly
// identifies cross-device link errors.
func TestIsCrossDeviceError(t *testing.T) {
	for _, tt := range testIsCrossDeviceErrorTestCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCrossDeviceError(tt.err))
		})
	}
}

// testDispatchAction defines the structure for test cases of the dispatchAction function,
// containing the action, and a check function for assertions on src and dst after dispatch.
type testDispatchAction struct {
	name   string
	action string
	check  func(t *testing.T, src, dst string)
	skip   func(t *testing.T, src, dst string) bool
}

// testDispatchActionTestCases defines a set of test cases for the dispatchAction function,
// covering move, copy, symlink, unknown action, and empty action scenarios.
var testDispatchActionTestCases = []testDispatchAction{
	{
		name:   "move removes src and creates dst",
		action: "move",
		check: func(t *testing.T, src, dst string) {
			assert.FileExists(t, dst)
			assert.NoFileExists(t, src)
		},
	},
	{
		name:   "copy keeps src and creates dst",
		action: "copy",
		check: func(t *testing.T, src, dst string) {
			assert.FileExists(t, dst)
			assert.FileExists(t, src)
			srcData, _ := os.ReadFile(src)
			dstData, _ := os.ReadFile(dst)
			assert.Equal(t, srcData, dstData)
		},
	},
	{
		name:   "symlink creates symlink at dst",
		action: "symlink",
		skip: func(t *testing.T, src, dst string) bool {
			if err := os.Symlink(src, dst+"_probe"); err != nil {
				t.Logf("symlink not available: %v", err)
				return true
			}
			os.Remove(dst + "_probe")
			return false
		},
		check: func(t *testing.T, src, dst string) {
			info, err := os.Lstat(dst)
			require.NoError(t, err)
			assert.True(t, info.Mode()&os.ModeSymlink != 0, "dst should be a symlink")
		},
	},
	{
		name:   "unknown action defaults to move",
		action: "unknown_action",
		check: func(t *testing.T, src, dst string) {
			assert.FileExists(t, dst)
			assert.NoFileExists(t, src)
		},
	},
	{
		name:   "empty action defaults to move",
		action: "",
		check: func(t *testing.T, src, dst string) {
			assert.FileExists(t, dst)
			assert.NoFileExists(t, src)
		},
	},
}

// TestDispatchAction tests the dispatchAction function with all action types
// to ensure it correctly dispatches move, copy, symlink, and fallback operations.
func TestDispatchAction(t *testing.T) {
	for _, tt := range testDispatchActionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "file.txt")
			dst := filepath.Join(t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

			if tt.skip != nil && tt.skip(t, src, dst) {
				t.Skip("action not available on this platform")
			}

			require.NoError(t, dispatchAction(context.Background(), tt.action, src, dst))
			tt.check(t, src, dst)
		})
	}
}

func newTestLogger() *pterm.Logger {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &l
}

func newTestMoveContext() MoveContext {
	return MoveContext{
		Logger: newTestLogger(),
	}
}
