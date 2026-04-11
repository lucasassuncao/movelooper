package cmd

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

// --- Regex filter integration ---

func TestRunMove_RegexFilter_OnlyMatchingFilesMoved(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "report_2024.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "invoice.pdf"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	// Only move files matching "report"
	cat.Source.Filter.Regex = "report"
	cat.Source.Filter.CompiledRegex = mustCompileRegex(t, "(?i)report")

	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "report_2024.pdf"))
	assert.FileExists(t, filepath.Join(src, "invoice.pdf"))
	assert.NoFileExists(t, filepath.Join(dst, "invoice.pdf"))
}

// --- Glob filter integration ---

func TestRunMove_GlobFilter_OnlyMatchingFilesMoved(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "IMG_001.jpg"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("x"), 0644))

	cat := buildCategory("Images", src, dst, []string{"jpg"})
	cat.Source.Filter.Glob = "IMG_*"

	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "IMG_001.jpg"))
	assert.FileExists(t, filepath.Join(src, "photo.jpg"))
}

// --- Ignore patterns integration ---

func TestRunMove_IgnorePattern_SkipsIgnoredFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "keep.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "temp_file.pdf"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	cat.Source.Filter.Ignore = []string{"temp_*"}

	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "keep.pdf"))
	assert.FileExists(t, filepath.Join(src, "temp_file.pdf"))
	assert.NoFileExists(t, filepath.Join(dst, "temp_file.pdf"))
}

// --- Size filter integration ---

func TestRunMove_MinSizeFilter_SkipsSmallFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	smallContent := make([]byte, 512)
	largeContent := make([]byte, 2048)
	require.NoError(t, os.WriteFile(filepath.Join(src, "small.txt"), smallContent, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "large.txt"), largeContent, 0644))

	cat := buildCategory("Texts", src, dst, []string{"txt"})
	cat.Source.Filter.MinSize = "1 KB"
	cat.Source.Filter.MinSizeBytes = 1024

	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "large.txt"))
	assert.FileExists(t, filepath.Join(src, "small.txt"))
}

// --- Age filter integration ---

func TestRunMove_MinAgeFilter_SkipsRecentFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	oldPath := filepath.Join(src, "old.txt")
	newPath := filepath.Join(src, "new.txt")
	require.NoError(t, os.WriteFile(oldPath, []byte("old"), 0644))
	require.NoError(t, os.WriteFile(newPath, []byte("new"), 0644))

	// Make old.txt appear to be 2 hours old
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldPath, twoHoursAgo, twoHoursAgo))

	cat := buildCategory("Texts", src, dst, []string{"txt"})
	cat.Source.Filter.MinAge = 1 * time.Hour

	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "old.txt"))
	assert.FileExists(t, filepath.Join(src, "new.txt"))
}

// --- Multi-extension category ---

func TestRunMove_MultipleExtensionsInOneCategory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("j"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "image.png"), []byte("p"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("d"), 0644))

	cat := buildCategory("Media", src, dst, []string{"jpg", "png"})
	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "photo.jpg"))
	assert.FileExists(t, filepath.Join(dst, "image.png"))
	assert.FileExists(t, filepath.Join(src, "doc.pdf"))
}

// --- Wildcard "all" extension ---

func TestRunMove_AllExtension_MovesEverything(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "a.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "b.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "c.zip"), []byte("x"), 0644))

	cat := buildCategory("All", src, dst, []string{"all"})
	m := newSilentMovelooper([]*models.Category{cat})
	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "a.pdf"))
	assert.FileExists(t, filepath.Join(dst, "b.txt"))
	assert.FileExists(t, filepath.Join(dst, "c.zip"))
}

// --- show-files flag (dry-run + show-files) ---

func TestRunMove_ShowFiles_DryRun(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "file.pdf"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	// showFiles=true with dryRun=true should not move anything and not error
	require.NoError(t, runMove(m, true, true))
	assert.FileExists(t, filepath.Join(src, "file.pdf"))
}

// --- validateDirectories warns but does not fail ---

func TestValidateDirectories_MissingDirsNoError(t *testing.T) {
	cat := buildCategory("Ghost", "/nonexistent/src", "/nonexistent/dst", []string{"txt"})
	m := newSilentMovelooper([]*models.Category{cat})

	// Should only warn, never panic or error
	assert.NotPanics(t, func() { validateDirectories(m) })
}

// --- resolveConfigPath ---

func TestResolveConfigPath_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0644))

	resolved, err := resolveConfigPath(path)
	require.NoError(t, err)
	assert.Equal(t, path, resolved)
}

func TestResolveConfigPath_ExplicitPathNotFound(t *testing.T) {
	_, err := resolveConfigPath("/nonexistent/path/movelooper.yaml")
	assert.Error(t, err)
}

// --- helpers ---

func mustCompileRegex(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(pattern)
	require.NoError(t, err)
	return re
}
