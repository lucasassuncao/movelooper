package cmd

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
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
			require.NoError(t, runMove(context.Background(), m, dryRun, showFiles, "", false))
			if tt.check != nil {
				tt.check(t, src, dst)
			}
		})
	}
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
			resolved, err := config.ResolveConfigPath(tt.path(t))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, resolved)
		})
	}
}

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

			require.NoError(t, runMove(context.Background(), m, tt.dryRun, false, "", false))
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

	require.NoError(t, runMove(context.Background(), m, false, false, "", false))

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

	require.NoError(t, runMove(context.Background(), m, false, false, "", false))

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

// fileExists is a nil-safe helper to check file existence.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// --- Action + rename integration tests ---

func TestRunMove_CopyAction(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "report.pdf"), []byte("x"), 0644))

	cat := buildCategory("docs", src, dst, []string{"pdf"})
	cat.Destination.Action = "copy"

	mctx := fileops.MoveContext{Logger: pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)}
	files, err := fileops.ReadDirectory(src)
	require.NoError(t, err)
	moved := fileops.MoveFiles(context.Background(), mctx, cat, files, "pdf", "batch_test")

	require.Len(t, moved, 1)
	assert.FileExists(t, filepath.Join(dst, "report.pdf"))
	assert.FileExists(t, filepath.Join(src, "report.pdf"))
}

func TestRunMove_CopyWithRename(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "photo.jpg"), []byte("x"), 0644))

	cat := buildCategory("images", src, dst, []string{"jpg"})
	cat.Destination.Action = "copy"
	cat.Destination.Rename = "{category}_{name}.{ext}"

	mctx := fileops.MoveContext{Logger: pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)}
	files, err := fileops.ReadDirectory(src)
	require.NoError(t, err)
	moved := fileops.MoveFiles(context.Background(), mctx, cat, files, "jpg", "batch_test")

	require.Len(t, moved, 1)
	assert.FileExists(t, filepath.Join(dst, "images_photo.jpg"))
	assert.FileExists(t, filepath.Join(src, "photo.jpg"))
}

func TestRunMove_SymlinkWithConflictRename(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "file.txt"), []byte("y"), 0644))

	cat := buildCategory("docs", src, dst, []string{"txt"})
	cat.Destination.Action = "symlink"
	cat.Destination.ConflictStrategy = "rename"

	mctx := fileops.MoveContext{Logger: pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)}
	files, err := fileops.ReadDirectory(src)
	require.NoError(t, err)
	moved := fileops.MoveFiles(context.Background(), mctx, cat, files, "txt", "batch_test")

	if len(moved) == 0 {
		t.Skip("symlink not available (likely missing privilege on Windows)")
	}
	assert.FileExists(t, filepath.Join(dst, "file.txt"))
	_, err = os.Lstat(filepath.Join(dst, "file(1).txt"))
	assert.NoError(t, err, "renamed symlink should exist")
}

// --- init --scan integration tests ---

func TestRunInit_Scan_GeneratesConfig(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "setup.exe"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unknown.xyz"), []byte("x"), 0644))

	outFile := filepath.Join(dir, "movelooper.yaml")
	opts := initOptions{
		scan:   dir,
		output: outFile,
	}

	err := runInit(opts)
	require.NoError(t, err)
	require.FileExists(t, outFile)

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)
	content := string(raw)

	assert.Contains(t, content, "images")
	assert.Contains(t, content, "jpg")
	assert.Contains(t, content, "videos")
	assert.Contains(t, content, "mp4")
	assert.Contains(t, content, "installers")
	assert.Contains(t, content, "hash_check")
	assert.Contains(t, content, "everything-else")
	assert.Contains(t, content, "enabled: false")
	assert.NotContains(t, content, "xyz")
}

func TestRunInit_Scan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "movelooper.yaml")
	opts := initOptions{
		scan:   dir,
		output: outFile,
	}

	err := runInit(opts)
	require.NoError(t, err)
	require.FileExists(t, outFile)

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)
	content := string(raw)

	assert.Contains(t, content, "everything-else")
	assert.NotContains(t, content, "images")
}

func TestRunInit_Scan_PathDoesNotExist(t *testing.T) {
	opts := initOptions{
		scan:   "/nonexistent/path/xyz",
		output: filepath.Join(t.TempDir(), "out.yaml"),
	}
	err := runInit(opts)
	assert.Error(t, err)
}

func TestRunInit_Scan_ExistingOutputNoForce(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(outFile, []byte("existing"), 0644))

	opts := initOptions{
		scan:   dir,
		output: outFile,
		force:  false,
	}
	// Should not overwrite — runInit returns nil but prints error (existing behavior).
	err := runInit(opts)
	require.NoError(t, err)

	raw, _ := os.ReadFile(outFile)
	assert.Equal(t, "existing", string(raw))
}

// --- Hook integration tests ---

// hookFailCmd returns a shell command that always exits with code 1.
func hookFailCmd() string {
	if runtime.GOOS == "windows" {
		return "exit /b 1"
	}
	return "exit 1"
}

func TestRunMove_Hooks(t *testing.T) {
	t.Run("before hook abort skips category on failure", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "a.pdf"), []byte("x"), 0644))

		cat := buildCategory("docs", src, dst, []string{"pdf"})
		cat.Hooks = &models.CategoryHooks{
			Before: &models.CategoryHook{
				OnFailure: "abort",
				Run:       []string{hookFailCmd()},
			},
		}

		m := newSilentMovelooper([]*models.Category{cat})
		err := runMove(context.Background(), m, false, false, "", false)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(src, "a.pdf"))
		assert.NoFileExists(t, filepath.Join(dst, "a.pdf"))
	})

	t.Run("after hook warn does not prevent move", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "b.pdf"), []byte("x"), 0644))

		cat := buildCategory("docs", src, dst, []string{"pdf"})
		cat.Hooks = &models.CategoryHooks{
			After: &models.CategoryHook{
				OnFailure: "warn",
				Run:       []string{hookFailCmd()},
			},
		}

		m := newSilentMovelooper([]*models.Category{cat})
		err := runMove(context.Background(), m, false, false, "", false)
		require.NoError(t, err)

		assert.NoFileExists(t, filepath.Join(src, "b.pdf"))
		assert.FileExists(t, filepath.Join(dst, "b.pdf"))
	})

	t.Run("before hook success then files are moved", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "c.pdf"), []byte("x"), 0644))

		cat := buildCategory("docs", src, dst, []string{"pdf"})
		cat.Hooks = &models.CategoryHooks{
			Before: &models.CategoryHook{
				OnFailure: "abort",
				Run:       []string{"echo before"},
			},
		}

		m := newSilentMovelooper([]*models.Category{cat})
		err := runMove(context.Background(), m, false, false, "", false)
		require.NoError(t, err)

		assert.NoFileExists(t, filepath.Join(src, "c.pdf"))
		assert.FileExists(t, filepath.Join(dst, "c.pdf"))
	})
}
