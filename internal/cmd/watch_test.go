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

// --- matchesExtensionAndFilters ---

func TestMatchesExtensionAndFilters(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		exts      []string
		setupCat  func(cat *models.Category)
		wantMatch bool
		noFile    bool
	}{
		{
			name:      "matches extension",
			fileName:  "report.pdf",
			exts:      []string{"pdf"},
			wantMatch: true,
		},
		{
			name:      "wrong extension",
			fileName:  "notes.txt",
			exts:      []string{"pdf"},
			wantMatch: false,
		},
		{
			name:      "non-existent file",
			fileName:  "ghost.pdf",
			exts:      []string{"pdf"},
			noFile:    true,
			wantMatch: false,
		},
		{
			name:     "regex filter matches",
			fileName: "report_2024.pdf",
			exts:     []string{"pdf"},
			setupCat: func(cat *models.Category) {
				cat.Source.Filter.Regex = "report"
				cat.Source.Filter.CompiledRegex = regexp.MustCompile("(?i)report")
			},
			wantMatch: true,
		},
		{
			name:     "regex filter no match",
			fileName: "invoice.pdf",
			exts:     []string{"pdf"},
			setupCat: func(cat *models.Category) {
				cat.Source.Filter.Regex = "report"
				cat.Source.Filter.CompiledRegex = regexp.MustCompile("(?i)report")
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			var filePath string
			if tt.noFile {
				filePath = "/nonexistent/" + tt.fileName
			} else {
				filePath = filepath.Join(src, tt.fileName)
				require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
			}

			cat := buildCategory("cat", src, t.TempDir(), tt.exts)
			if tt.setupCat != nil {
				tt.setupCat(cat)
			}

			assert.Equal(t, tt.wantMatch, matchesExtensionAndFilters(cat, tt.fileName, filePath))
		})
	}
}

// --- resolveDryRunDest ---

func TestResolveDryRunDest(t *testing.T) {
	tests := []struct {
		name       string
		organizeBy string
		fileName   string
		fileExists bool
		wantSuffix string // expected suffix appended to dst (empty = dst itself)
	}{
		{
			name:       "no template returns dst",
			wantSuffix: "",
		},
		{
			name:       "ext template appends subdir",
			organizeBy: "{ext}",
			fileName:   "photo.jpg",
			fileExists: true,
			wantSuffix: "jpg",
		},
		{
			name:       "non-existent file falls back to dst",
			organizeBy: "{ext}",
			fileName:   "file.pdf",
			fileExists: false,
			wantSuffix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()

			fileName := tt.fileName
			if fileName == "" {
				fileName = "file.pdf"
			}
			filePath := filepath.Join(src, fileName)

			if tt.fileExists {
				require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
			} else if !tt.fileExists && tt.organizeBy != "" {
				filePath = "/nonexistent/" + fileName
			}

			cat := &models.Category{
				Name:    "cat",
				Enabled: boolPtr(true),
				Source:  models.CategorySource{Path: src, Extensions: []string{"pdf", "jpg"}},
				Destination: models.CategoryDestination{
					Path:       dst,
					OrganizeBy: tt.organizeBy,
				},
			}

			result := resolveDryRunDest(cat, filePath)
			if tt.wantSuffix == "" {
				assert.Equal(t, dst, result)
			} else {
				assert.Equal(t, filepath.Join(dst, tt.wantSuffix), result)
			}
		})
	}
}

// --- attemptMoveFile ---

func TestAttemptMoveFile(t *testing.T) {
	tests := []struct {
		name      string
		fileName  string
		catExts   []string
		wrongSrc  bool // file in different src than category
		dryRun    bool
		wantMoved bool
	}{
		{
			name:      "dry-run does not move",
			fileName:  "doc.pdf",
			catExts:   []string{"pdf"},
			dryRun:    true,
			wantMoved: false,
		},
		{
			name:      "no matching category stays",
			fileName:  "notes.txt",
			catExts:   []string{"pdf"},
			wantMoved: false,
		},
		{
			name:      "moves matching file",
			fileName:  "report.pdf",
			catExts:   []string{"pdf"},
			wantMoved: true,
		},
		{
			name:      "ignores file from wrong source dir",
			fileName:  "file.pdf",
			catExts:   []string{"pdf"},
			wrongSrc:  true,
			wantMoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catSrc := t.TempDir()
			dst := t.TempDir()

			var fileSrc string
			if tt.wrongSrc {
				fileSrc = t.TempDir()
			} else {
				fileSrc = catSrc
			}

			filePath := filepath.Join(fileSrc, tt.fileName)
			require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))

			cat := buildCategory("cat", catSrc, dst, tt.catExts)
			m := newSilentMovelooper([]*models.Category{cat})

			err := attemptMoveFile(m, filePath, tt.dryRun)
			assert.NoError(t, err)

			if tt.wantMoved {
				assert.NoFileExists(t, filePath)
				assert.FileExists(t, filepath.Join(dst, tt.fileName))
			} else {
				assert.FileExists(t, filePath)
			}
		})
	}
}

// --- performInitialScan ---

func TestPerformInitialScan(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		ext       []string
		disabled  bool
		ignore    []string
		wantFiles []string
	}{
		{
			name:      "adds matching files",
			files:     []string{"a.pdf", "b.txt"},
			ext:       []string{"pdf"},
			wantFiles: []string{"a.pdf"},
		},
		{
			name:      "skips disabled category",
			files:     []string{"a.pdf"},
			ext:       []string{"pdf"},
			disabled:  true,
			wantFiles: nil,
		},
		{
			name:      "ignores ignored files",
			files:     []string{"ignore_me.pdf", "keep.pdf"},
			ext:       []string{"pdf"},
			ignore:    []string{"ignore_*"},
			wantFiles: []string{"keep.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := t.TempDir()
			dst := t.TempDir()

			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(src, f), []byte("x"), 0644))
			}

			cat := buildCategory("cat", src, dst, tt.ext)
			if tt.disabled {
				cat.Enabled = boolPtr(false)
			}
			cat.Source.Filter.Ignore = tt.ignore

			m := newSilentMovelooper([]*models.Category{cat})
			tracker := &fileTracker{files: make(map[string]time.Time)}
			performInitialScan(m, tracker)

			tracker.mu.Lock()
			defer tracker.mu.Unlock()

			assert.Len(t, tracker.files, len(tt.wantFiles))
			for _, f := range tt.wantFiles {
				assert.Contains(t, tracker.files, filepath.Join(src, f))
			}
		})
	}
}

// --- processPendingFiles ---

// buildStaleTracker creates a src PDF file aged 10 minutes and returns a tracker
// with that file already registered as stale. Use dst to verify move outcomes.
func buildStaleTracker(t *testing.T, name string) (m *models.Movelooper, dst, filePath string, tracker *fileTracker) {
	t.Helper()
	src := t.TempDir()
	dst = t.TempDir()
	filePath = filepath.Join(src, name)
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
	oldTime := time.Now().Add(-10 * time.Minute)
	require.NoError(t, os.Chtimes(filePath, oldTime, oldTime))
	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m = newSilentMovelooper([]*models.Category{cat})
	tracker = &fileTracker{files: map[string]time.Time{
		filePath: time.Now().Add(-10 * time.Minute),
	}}
	return
}

func TestProcessPendingFiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) (m *models.Movelooper, dst string, filePath string, tracker *fileTracker)
		threshold time.Duration
		dryRun    bool
		wantMoved bool
		wantGone  bool // file removed from tracker (e.g. deleted file)
	}{
		{
			name: "moves stable file",
			setup: func(t *testing.T) (*models.Movelooper, string, string, *fileTracker) {
				return buildStaleTracker(t, "old.pdf")
			},
			threshold: 5 * time.Minute,
			wantMoved: true,
		},
		{
			name: "skips fresh file",
			setup: func(t *testing.T) (*models.Movelooper, string, string, *fileTracker) {
				src := t.TempDir()
				dst := t.TempDir()
				filePath := filepath.Join(src, "fresh.pdf")
				require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
				cat := buildCategory("PDFs", src, dst, []string{"pdf"})
				m := newSilentMovelooper([]*models.Category{cat})
				tracker := &fileTracker{files: map[string]time.Time{filePath: time.Now()}}
				return m, dst, filePath, tracker
			},
			threshold: 5 * time.Minute,
			wantMoved: false,
		},
		{
			name: "removes deleted file from tracker",
			setup: func(t *testing.T) (*models.Movelooper, string, string, *fileTracker) {
				src := t.TempDir()
				dst := t.TempDir()
				ghostPath := filepath.Join(src, "ghost.pdf")
				cat := buildCategory("PDFs", src, dst, []string{"pdf"})
				m := newSilentMovelooper([]*models.Category{cat})
				tracker := &fileTracker{files: map[string]time.Time{ghostPath: time.Now().Add(-10 * time.Minute)}}
				return m, dst, ghostPath, tracker
			},
			threshold: 5 * time.Minute,
			wantGone:  true,
		},
		{
			name: "dry-run does not move",
			setup: func(t *testing.T) (*models.Movelooper, string, string, *fileTracker) {
				return buildStaleTracker(t, "stable.pdf")
			},
			threshold: 5 * time.Minute,
			dryRun:    true,
			wantMoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, dst, filePath, tracker := tt.setup(t)
			processPendingFiles(m, tracker, tt.threshold, tt.dryRun)

			fileName := filepath.Base(filePath)
			switch {
			case tt.wantMoved:
				assert.NoFileExists(t, filePath)
				assert.FileExists(t, filepath.Join(dst, fileName))
			case tt.wantGone:
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				assert.NotContains(t, tracker.files, filePath)
			default:
				assert.FileExists(t, filePath)
			}
		})
	}
}
