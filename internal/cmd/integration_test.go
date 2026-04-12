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

// --- Filter integration tests ---

func TestRunMove_Filters(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, src, dst string)
		buildCat func(src, dst string) *models.Category
		check    func(t *testing.T, src, dst string)
	}{
		{
			name: "regex filter moves only matching files",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "report_2024.pdf"), []byte("x"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "invoice.pdf"), []byte("x"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				cat := buildCategory("PDFs", src, dst, []string{"pdf"})
				cat.Source.Filter.Regex = "report"
				cat.Source.Filter.CompiledRegex = regexp.MustCompile("(?i)report")
				return cat
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "report_2024.pdf"))
				assert.FileExists(t, filepath.Join(src, "invoice.pdf"))
				assert.NoFileExists(t, filepath.Join(dst, "invoice.pdf"))
			},
		},
		{
			name: "glob filter moves only matching files",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "IMG_001.jpg"), []byte("x"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("x"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				cat := buildCategory("Images", src, dst, []string{"jpg"})
				cat.Source.Filter.Glob = "IMG_*"
				return cat
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "IMG_001.jpg"))
				assert.FileExists(t, filepath.Join(src, "photo.jpg"))
			},
		},
		{
			name: "ignore pattern skips ignored files",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "keep.pdf"), []byte("x"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "temp_file.pdf"), []byte("x"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				cat := buildCategory("PDFs", src, dst, []string{"pdf"})
				cat.Source.Filter.Ignore = []string{"temp_*"}
				return cat
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "keep.pdf"))
				assert.FileExists(t, filepath.Join(src, "temp_file.pdf"))
				assert.NoFileExists(t, filepath.Join(dst, "temp_file.pdf"))
			},
		},
		{
			name: "min size filter skips small files",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "small.txt"), make([]byte, 512), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "large.txt"), make([]byte, 2048), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				cat := buildCategory("Texts", src, dst, []string{"txt"})
				cat.Source.Filter.MinSize = "1 KB"
				cat.Source.Filter.MinSizeBytes = 1024
				return cat
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "large.txt"))
				assert.FileExists(t, filepath.Join(src, "small.txt"))
			},
		},
		{
			name: "min age filter skips recent files",
			setup: func(t *testing.T, src, dst string) {
				oldPath := filepath.Join(src, "old.txt")
				require.NoError(t, os.WriteFile(oldPath, []byte("old"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0644))
				twoHoursAgo := time.Now().Add(-2 * time.Hour)
				require.NoError(t, os.Chtimes(oldPath, twoHoursAgo, twoHoursAgo))
			},
			buildCat: func(src, dst string) *models.Category {
				cat := buildCategory("Texts", src, dst, []string{"txt"})
				cat.Source.Filter.MinAge = 1 * time.Hour
				return cat
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "old.txt"))
				assert.FileExists(t, filepath.Join(src, "new.txt"))
			},
		},
		{
			name: "multiple extensions in one category",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("j"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "image.png"), []byte("p"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "doc.pdf"), []byte("d"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				return buildCategory("Media", src, dst, []string{"jpg", "png"})
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "photo.jpg"))
				assert.FileExists(t, filepath.Join(dst, "image.png"))
				assert.FileExists(t, filepath.Join(src, "doc.pdf"))
			},
		},
		{
			name: "all extension moves everything",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "a.pdf"), []byte("x"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "b.txt"), []byte("x"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(src, "c.zip"), []byte("x"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				return buildCategory("All", src, dst, []string{"all"})
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(dst, "a.pdf"))
				assert.FileExists(t, filepath.Join(dst, "b.txt"))
				assert.FileExists(t, filepath.Join(dst, "c.zip"))
			},
		},
		{
			name: "show-files dry-run does not move",
			setup: func(t *testing.T, src, dst string) {
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.pdf"), []byte("x"), 0644))
			},
			buildCat: func(src, dst string) *models.Category {
				return buildCategory("PDFs", src, dst, []string{"pdf"})
			},
			check: func(t *testing.T, src, dst string) {
				assert.FileExists(t, filepath.Join(src, "file.pdf"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()
			tt.setup(t, src, dst)
			cat := tt.buildCat(src, dst)
			m := newSilentMovelooper([]*models.Category{cat})

			dryRun := tt.name == "show-files dry-run does not move"
			showFiles := dryRun
			require.NoError(t, runMove(m, dryRun, showFiles))
			if tt.check != nil {
				tt.check(t, src, dst)
			}
		})
	}
}

// --- validateDirectories ---

func TestValidateDirectories_MissingDirsNoError(t *testing.T) {
	cat := buildCategory("Ghost", "/nonexistent/src", "/nonexistent/dst", []string{"txt"})
	m := newSilentMovelooper([]*models.Category{cat})
	assert.NotPanics(t, func() { validateDirectories(m) })
}

// --- resolveConfigPath ---

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "explicit path returns path",
			path: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "movelooper.yaml")
				require.NoError(t, os.WriteFile(p, []byte(""), 0644))
				return p
			},
		},
		{
			name:    "explicit path not found returns error",
			path:    func(t *testing.T) string { return "/nonexistent/path/movelooper.yaml" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolveConfigPath(tt.path(t))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, resolved)
		})
	}
}
