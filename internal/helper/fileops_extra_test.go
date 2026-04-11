package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- applyConflictStrategy ---

func TestApplyConflictStrategy_NoConflict(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("data"), 0644))

	// dstFile does not exist — no conflict
	resolved, skip := applyConflictStrategy(newTestMoveContext(), "rename", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_SkipStrategy(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("existing"), 0644))

	_, skip := applyConflictStrategy(newTestMoveContext(), "skip", srcFile, dstFile, dst, "file.txt")
	assert.True(t, skip)
	// Source must remain untouched
	assert.FileExists(t, srcFile)
}

func TestApplyConflictStrategy_RenameStrategy(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("existing"), 0644))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "rename", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Contains(t, resolved, "(1)")
}

func TestApplyConflictStrategy_OverwriteStrategy(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("existing"), 0644))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "overwrite", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_HashCheck_Duplicate(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	content := []byte("identical content")
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, content, 0644))
	require.NoError(t, os.WriteFile(dstFile, content, 0644))

	_, skip := applyConflictStrategy(newTestMoveContext(), "hash_check", srcFile, dstFile, dst, "file.txt")
	assert.True(t, skip)
	// Source removed (duplicate)
	assert.NoFileExists(t, srcFile)
}

func TestApplyConflictStrategy_NewestStrategy_SrcNewer(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("new"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("old"), 0644))

	// Make dst clearly older than src
	oldTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(dstFile, oldTime, oldTime))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "newest", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_OldestStrategy_SrcOlder(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("old"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("new"), 0644))

	// Make src clearly older than dst so "oldest" keeps src (moves it to dst)
	oldTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(srcFile, oldTime, oldTime))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "oldest", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_LargerStrategy_SrcLarger(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("larger content here"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("small"), 0644))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "larger", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_SmallerStrategy_SrcSmaller(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("tiny"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("much larger content here"), 0644))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "smaller", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Equal(t, dstFile, resolved)
}

func TestApplyConflictStrategy_UnknownFallsToRename(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("x"), 0644))
	require.NoError(t, os.WriteFile(dstFile, []byte("y"), 0644))

	resolved, skip := applyConflictStrategy(newTestMoveContext(), "does_not_exist", srcFile, dstFile, dst, "file.txt")
	assert.False(t, skip)
	assert.Contains(t, resolved, "(1)")
}

// --- isCrossDeviceError ---

func TestIsCrossDeviceError_NonLinkError(t *testing.T) {
	err := os.ErrPermission
	assert.False(t, isCrossDeviceError(err))
}

func TestIsCrossDeviceError_NilError(t *testing.T) {
	assert.False(t, isCrossDeviceError(nil))
}

func TestIsCrossDeviceError_LinkErrorWithPermission(t *testing.T) {
	linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrPermission}
	assert.False(t, isCrossDeviceError(linkErr))
}

// --- GenerateLogArgs ---

func TestGenerateLogArgs_ReturnsNamePairs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), []byte("x"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	args := GenerateLogArgs(entries, "pdf")
	// Each match produces a "name", "<filename>" pair
	assert.Len(t, args, 4) // 2 pdfs × 2 elements each
	assert.Equal(t, "name", args[0])
	assert.Equal(t, "name", args[2])
}

func TestGenerateLogArgs_NoMatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	args := GenerateLogArgs(entries, "pdf")
	assert.Empty(t, args)
}

func TestGenerateLogArgs_AllExtension(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	args := GenerateLogArgs(entries, "all")
	assert.Len(t, args, 4) // 2 files × 2 elements each
}

// --- MoveFiles with no DefaultConflictStrategy ---

func TestMoveFiles_DefaultsToRenameWhenNoStrategy(t *testing.T) {
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
			ConflictStrategy: "", // empty — should default to rename
		},
	}

	ctx := newTestMoveContext()
	moved := MoveFiles(ctx, category, entries, "txt", "batch_default")

	assert.Len(t, moved, 1)
	assert.FileExists(t, filepath.Join(dst, "file(1).txt"))
}
