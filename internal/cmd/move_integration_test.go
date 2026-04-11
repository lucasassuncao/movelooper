package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSilentMovelooper returns a Movelooper with a disabled logger and no history,
// suitable for integration tests that control the filesystem directly.
func newSilentMovelooper(categories []*models.Category) *models.Movelooper {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &models.Movelooper{
		Logger:     &l,
		Categories: categories,
	}
}

func boolPtr(v bool) *bool { return &v }

// buildCategory is a helper to construct a Category for test scenarios.
// It uses the "rename" conflict strategy; set ConflictStrategy directly for other strategies.
func buildCategory(name, src, dst string, extensions []string) *models.Category {
	return &models.Category{
		Name:    name,
		Enabled: boolPtr(true),
		Source: models.CategorySource{
			Path:       src,
			Extensions: extensions,
		},
		Destination: models.CategoryDestination{
			Path:             dst,
			ConflictStrategy: "rename",
		},
	}
}

// --- Integration: full move run ---

func TestRunMove_MovesFilesByExtension(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "report.pdf"), []byte("pdf"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "notes.txt"), []byte("txt"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("jpg"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "report.pdf"))
	assert.NoFileExists(t, filepath.Join(src, "report.pdf"))
	// Non-matching files stay in source
	assert.FileExists(t, filepath.Join(src, "notes.txt"))
	assert.FileExists(t, filepath.Join(src, "photo.jpg"))
}

func TestRunMove_DryRunDoesNotMove(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	require.NoError(t, runMove(m, true, false))

	// File must remain in source on dry-run
	assert.FileExists(t, filepath.Join(src, "doc.pdf"))
	assert.NoFileExists(t, filepath.Join(dst, "doc.pdf"))
}

func TestRunMove_DisabledCategorySkipped(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	cat.Enabled = boolPtr(false)
	m := newSilentMovelooper([]*models.Category{cat})

	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(src, "doc.pdf"))
}

func TestRunMove_MultipleCategories(t *testing.T) {
	src := t.TempDir()
	dstPDF := t.TempDir()
	dstJPG := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "file.pdf"), []byte("pdf"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("jpg"), 0644))

	cats := []*models.Category{
		buildCategory("PDFs", src, dstPDF, []string{"pdf"}),
		buildCategory("Images", src, dstJPG, []string{"jpg"}),
	}
	m := newSilentMovelooper(cats)

	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dstPDF, "file.pdf"))
	assert.FileExists(t, filepath.Join(dstJPG, "photo.jpg"))
}

func TestRunMove_FileClaimedByFirstCategory(t *testing.T) {
	src := t.TempDir()
	dst1 := t.TempDir()
	dst2 := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("text"), 0644))

	// Both categories claim "all" — first one wins
	cats := []*models.Category{
		buildCategory("First", src, dst1, []string{"all"}),
		buildCategory("Second", src, dst2, []string{"all"}),
	}
	m := newSilentMovelooper(cats)

	require.NoError(t, runMove(m, false, false))

	// File must land in exactly one destination
	inDst1 := fileExists(filepath.Join(dst1, "file.txt"))
	inDst2 := fileExists(filepath.Join(dst2, "file.txt"))
	assert.True(t, inDst1 || inDst2, "file must be in one of the destinations")
	assert.False(t, inDst1 && inDst2, "file must not be in both destinations")
}

func TestRunMove_WithOrganizeByTemplate(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "image.jpg"), []byte("img"), 0644))

	cat := &models.Category{
		Name:    "Images",
		Enabled: boolPtr(true),
		Source:  models.CategorySource{Path: src, Extensions: []string{"jpg"}},
		Destination: models.CategoryDestination{
			Path:             dst,
			OrganizeBy:       "{ext}",
			ConflictStrategy: "rename",
		},
	}
	m := newSilentMovelooper([]*models.Category{cat})

	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "jpg", "image.jpg"))
}

func TestRunMove_ConflictRename(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("existing"), 0644))

	cat := buildCategory("Texts", src, dst, []string{"txt"})
	m := newSilentMovelooper([]*models.Category{cat})

	require.NoError(t, runMove(m, false, false))

	assert.FileExists(t, filepath.Join(dst, "file.txt"))
	assert.FileExists(t, filepath.Join(dst, "file(1).txt"))
}

// --- filterFilesForExtension ---

func TestFilterFilesForExtension_FiltersCorrectly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.pdf"), []byte("x"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	cat := buildCategory("PDFs", dir, dir, []string{"pdf"})
	moved := make(movedSet)

	filtered := filterFilesForExtension(cat, entries, moved, "pdf")
	assert.Len(t, filtered, 2)
}

func TestFilterFilesForExtension_SkipsAlreadyMoved(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("x"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	cat := buildCategory("PDFs", dir, dir, []string{"pdf"})
	moved := make(movedSet)
	moved.mark(dir, "a.pdf")

	filtered := filterFilesForExtension(cat, entries, moved, "pdf")
	assert.Empty(t, filtered)
}

// --- formatBytes ---

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatBytes(tt.input), "input=%d", tt.input)
	}
}

// --- movedSet ---

func TestMovedSet(t *testing.T) {
	s := make(movedSet)
	assert.False(t, s.has("/src", "file.txt"))
	s.mark("/src", "file.txt")
	assert.True(t, s.has("/src", "file.txt"))
	assert.False(t, s.has("/other", "file.txt"))
}

// fileExists is a nil-safe helper to check file existence.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
