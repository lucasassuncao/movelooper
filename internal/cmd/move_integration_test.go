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

func TestRunMove(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, src string, extraDst ...*string)
		cats   func(t *testing.T, src string, extraDst ...*string) []*models.Category
		dryRun bool
		check  func(t *testing.T, src string, extraDst ...*string)
	}{
		{
			name: "moves files by extension",
			setup: func(t *testing.T, src string, d ...*string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "report.pdf"), []byte("pdf"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "notes.txt"), []byte("txt"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("jpg"), 0644))
			},
			cats: func(t *testing.T, src string, d ...*string) []*models.Category {
				return []*models.Category{buildCategory("PDFs", src, *d[0], []string{"pdf"})}
			},
			check: func(t *testing.T, src string, d ...*string) {
				assert.FileExists(t, filepath.Join(*d[0], "report.pdf"))
				assert.NoFileExists(t, filepath.Join(src, "report.pdf"))
				assert.FileExists(t, filepath.Join(src, "notes.txt"))
				assert.FileExists(t, filepath.Join(src, "photo.jpg"))
			},
		},
		{
			name: "dry-run does not move",
			setup: func(t *testing.T, src string, d ...*string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))
			},
			cats: func(t *testing.T, src string, d ...*string) []*models.Category {
				return []*models.Category{buildCategory("PDFs", src, *d[0], []string{"pdf"})}
			},
			dryRun: true,
			check: func(t *testing.T, src string, d ...*string) {
				assert.FileExists(t, filepath.Join(src, "doc.pdf"))
				assert.NoFileExists(t, filepath.Join(*d[0], "doc.pdf"))
			},
		},
		{
			name: "disabled category skipped",
			setup: func(t *testing.T, src string, d ...*string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("pdf"), 0644))
			},
			cats: func(t *testing.T, src string, d ...*string) []*models.Category {
				cat := buildCategory("PDFs", src, *d[0], []string{"pdf"})
				cat.Enabled = boolPtr(false)
				return []*models.Category{cat}
			},
			check: func(t *testing.T, src string, d ...*string) {
				assert.FileExists(t, filepath.Join(src, "doc.pdf"))
			},
		},
		{
			name: "conflict rename",
			setup: func(t *testing.T, src string, d ...*string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("new"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(*d[0], "file.txt"), []byte("existing"), 0644))
			},
			cats: func(t *testing.T, src string, d ...*string) []*models.Category {
				return []*models.Category{buildCategory("Texts", src, *d[0], []string{"txt"})}
			},
			check: func(t *testing.T, src string, d ...*string) {
				assert.FileExists(t, filepath.Join(*d[0], "file.txt"))
				assert.FileExists(t, filepath.Join(*d[0], "file(1).txt"))
			},
		},
		{
			name: "organize by ext template",
			setup: func(t *testing.T, src string, d ...*string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "image.jpg"), []byte("img"), 0644))
			},
			cats: func(t *testing.T, src string, d ...*string) []*models.Category {
				return []*models.Category{{
					Name:    "Images",
					Enabled: boolPtr(true),
					Source:  models.CategorySource{Path: src, Extensions: []string{"jpg"}},
					Destination: models.CategoryDestination{
						Path:             *d[0],
						OrganizeBy:       "{ext}",
						ConflictStrategy: "rename",
					},
				}}
			},
			check: func(t *testing.T, src string, d ...*string) {
				assert.FileExists(t, filepath.Join(*d[0], "jpg", "image.jpg"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()
			dstRef := &dst

			tt.setup(t, src, dstRef)
			cats := tt.cats(t, src, dstRef)
			m := newSilentMovelooper(cats)

			require.NoError(t, runMove(m, tt.dryRun, false))
			if tt.check != nil {
				tt.check(t, src, dstRef)
			}
		})
	}
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

	cats := []*models.Category{
		buildCategory("First", src, dst1, []string{"all"}),
		buildCategory("Second", src, dst2, []string{"all"}),
	}
	m := newSilentMovelooper(cats)

	require.NoError(t, runMove(m, false, false))

	inDst1 := fileExists(filepath.Join(dst1, "file.txt"))
	inDst2 := fileExists(filepath.Join(dst2, "file.txt"))
	assert.True(t, inDst1 || inDst2, "file must be in one of the destinations")
	assert.False(t, inDst1 && inDst2, "file must not be in both destinations")
}

// --- filterFilesForExtension ---

func TestFilterFilesForExtension(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		ext       string
		preMarked []string // files to mark as already moved
		wantLen   int
	}{
		{
			name:    "filters correctly by extension",
			files:   []string{"a.pdf", "b.txt", "c.pdf"},
			ext:     "pdf",
			wantLen: 2,
		},
		{
			name:      "skips already moved files",
			files:     []string{"a.pdf"},
			ext:       "pdf",
			preMarked: []string{"a.pdf"},
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644))
			}

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)

			cat := buildCategory("PDFs", dir, dir, []string{tt.ext})
			moved := make(movedSet)
			for _, f := range tt.preMarked {
				moved.mark(dir, f)
			}

			filtered := filterFilesForExtension(cat, entries, moved, tt.ext)
			assert.Len(t, filtered, tt.wantLen)
		})
	}
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
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBytes(tt.input))
		})
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
