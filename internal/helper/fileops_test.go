package helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger returns a silent pterm logger suitable for unit tests.
func newTestLogger() *pterm.Logger {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &l
}

// newTestMoveContext returns a MoveContext with a silent logger and no history.
func newTestMoveContext() MoveContext {
	return MoveContext{Logger: newTestLogger()}
}

// --- CreateDirectory ---

func TestCreateDirectory_CreatesNew(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	require.NoError(t, CreateDirectory(dir))
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateDirectory_Idempotent(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, CreateDirectory(dir))
	assert.NoError(t, CreateDirectory(dir))
}

// --- ReadDirectory ---

func TestReadDirectory_ReturnsEntries(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644))

	entries, err := ReadDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestReadDirectory_NonExistentReturnsError(t *testing.T) {
	_, err := ReadDirectory(filepath.Join(t.TempDir(), "nonexistent"))
	assert.Error(t, err)
}

// --- copyFile ---

func TestCopyFile_CopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(src, content, 0644))

	require.NoError(t, copyFile(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestCopyFile_PreservesModTime(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))

	srcInfo, err := os.Stat(src)
	require.NoError(t, err)

	require.NoError(t, copyFile(src, dst))

	dstInfo, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, srcInfo.ModTime().Unix(), dstInfo.ModTime().Unix())
}

// --- MoveFiles ---

func TestMoveFiles_MovesMatchingExtension(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "img.jpg"), []byte("jpg"), 0644))

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	enabled := true
	category := &models.Category{
		Name:    "PDFs",
		Enabled: &enabled,
		Source:  models.CategorySource{Path: src, Extensions: []string{"pdf"}},
		Destination: models.CategoryDestination{
			Path:             dst,
			ConflictStrategy: "rename",
		},
	}

	ctx := newTestMoveContext()
	moved := MoveFiles(ctx, category, entries, "pdf", "batch_test")

	assert.Equal(t, []string{"doc.pdf"}, moved)
	assert.FileExists(t, filepath.Join(dst, "doc.pdf"))
	assert.NoFileExists(t, filepath.Join(src, "doc.pdf"))
	// jpg should stay untouched
	assert.FileExists(t, filepath.Join(src, "img.jpg"))
}

func TestMoveFiles_SkipsOnConflictSkipStrategy(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("existing"), 0644))

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	enabled := true
	category := &models.Category{
		Name:    "Texts",
		Enabled: &enabled,
		Source:  models.CategorySource{Path: src, Extensions: []string{"txt"}},
		Destination: models.CategoryDestination{
			Path:             dst,
			ConflictStrategy: "skip",
		},
	}

	ctx := newTestMoveContext()
	moved := MoveFiles(ctx, category, entries, "txt", "batch_skip")

	assert.Empty(t, moved)
	// Source should still be there (skipped)
	assert.FileExists(t, filepath.Join(src, "file.txt"))
}

func TestMoveFiles_WithOrganizeBy(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("img"), 0644))

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	enabled := true
	category := &models.Category{
		Name:    "Photos",
		Enabled: &enabled,
		Source:  models.CategorySource{Path: src, Extensions: []string{"jpg"}},
		Destination: models.CategoryDestination{
			Path:             dst,
			OrganizeBy:       "{ext}",
			ConflictStrategy: "rename",
		},
	}

	ctx := newTestMoveContext()
	moved := MoveFiles(ctx, category, entries, "jpg", "batch_org")

	assert.Equal(t, []string{"photo.jpg"}, moved)
	assert.FileExists(t, filepath.Join(dst, "jpg", "photo.jpg"))
}

func TestMoveFiles_ExtAllMovesAll(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "b.pdf"), []byte("b"), 0644))

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	enabled := true
	category := &models.Category{
		Name:    "All",
		Enabled: &enabled,
		Source:  models.CategorySource{Path: src},
		Destination: models.CategoryDestination{
			Path:             dst,
			ConflictStrategy: "rename",
		},
	}

	ctx := newTestMoveContext()
	moved := MoveFiles(ctx, category, entries, "all", "batch_all")

	assert.Len(t, moved, 2)
}
