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

// --- fileInfoDirEntry ---

func TestFileInfoDirEntry_Interface(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	info, err := os.Lstat(path)
	require.NoError(t, err)

	entry := fileInfoDirEntry{info: info}
	assert.Equal(t, "test.txt", entry.Name())
	assert.False(t, entry.IsDir())
	assert.True(t, entry.Type().IsRegular())

	got, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, info.Name(), got.Name())
}

func TestFileInfoDirEntry_Directory(t *testing.T) {
	dir := t.TempDir()
	info, err := os.Lstat(dir)
	require.NoError(t, err)

	entry := fileInfoDirEntry{info: info}
	assert.True(t, entry.IsDir())
	assert.False(t, entry.Type().IsRegular())
}

// --- matchesExtensionAndFilters ---

func TestMatchesExtensionAndFilters_Match(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.pdf")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0644))

	cat := buildCategory("PDFs", dir, t.TempDir(), []string{"pdf"})
	assert.True(t, matchesExtensionAndFilters(cat, "report.pdf", path))
}

func TestMatchesExtensionAndFilters_WrongExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0644))

	cat := buildCategory("PDFs", dir, t.TempDir(), []string{"pdf"})
	assert.False(t, matchesExtensionAndFilters(cat, "notes.txt", path))
}

func TestMatchesExtensionAndFilters_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	cat := buildCategory("PDFs", dir, t.TempDir(), []string{"pdf"})
	assert.False(t, matchesExtensionAndFilters(cat, "ghost.pdf", "/nonexistent/ghost.pdf"))
}

func TestMatchesExtensionAndFilters_RegexFilter(t *testing.T) {
	dir := t.TempDir()
	matchPath := filepath.Join(dir, "report_2024.pdf")
	noMatchPath := filepath.Join(dir, "invoice.pdf")
	require.NoError(t, os.WriteFile(matchPath, []byte("x"), 0644))
	require.NoError(t, os.WriteFile(noMatchPath, []byte("x"), 0644))

	cat := buildCategory("PDFs", dir, t.TempDir(), []string{"pdf"})
	cat.Source.Filter.Regex = "report"
	cat.Source.Filter.CompiledRegex = regexp.MustCompile("(?i)report")

	assert.True(t, matchesExtensionAndFilters(cat, "report_2024.pdf", matchPath))
	assert.False(t, matchesExtensionAndFilters(cat, "invoice.pdf", noMatchPath))
}

// --- resolveDryRunDest ---

func TestResolveDryRunDest_NoTemplate(t *testing.T) {
	dir := t.TempDir()
	dst := t.TempDir()
	cat := buildCategory("Docs", dir, dst, []string{"pdf"})
	result := resolveDryRunDest(cat, filepath.Join(dir, "file.pdf"))
	assert.Equal(t, dst, result)
}

func TestResolveDryRunDest_WithExtTemplate(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	filePath := filepath.Join(src, "photo.jpg")
	require.NoError(t, os.WriteFile(filePath, []byte("img"), 0644))

	cat := &models.Category{
		Name:    "Images",
		Enabled: boolPtr(true),
		Source:  models.CategorySource{Path: src, Extensions: []string{"jpg"}},
		Destination: models.CategoryDestination{
			Path:       dst,
			OrganizeBy: "{ext}",
		},
	}
	result := resolveDryRunDest(cat, filePath)
	assert.Equal(t, filepath.Join(dst, "jpg"), result)
}

func TestResolveDryRunDest_NonExistentFile(t *testing.T) {
	dst := t.TempDir()
	cat := buildCategory("Docs", t.TempDir(), dst, []string{"pdf"})
	cat.Destination.OrganizeBy = "{ext}"
	// File doesn't exist — should fall back to base dest
	result := resolveDryRunDest(cat, "/nonexistent/file.pdf")
	assert.Equal(t, dst, result)
}

// --- attemptMoveFile ---

func TestAttemptMoveFile_DryRun_Logs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	filePath := filepath.Join(src, "doc.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("pdf"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	err := attemptMoveFile(m, filePath, true)
	assert.NoError(t, err)
	// Dry-run: file must stay in source
	assert.FileExists(t, filePath)
	assert.NoFileExists(t, filepath.Join(dst, "doc.pdf"))
}

func TestAttemptMoveFile_NoMatchingCategory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	filePath := filepath.Join(src, "notes.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("txt"), 0644))

	// Category only matches pdf
	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	err := attemptMoveFile(m, filePath, false)
	assert.NoError(t, err)
	// No matching category — file stays
	assert.FileExists(t, filePath)
}

func TestAttemptMoveFile_MovesFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	filePath := filepath.Join(src, "report.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("pdf"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	err := attemptMoveFile(m, filePath, false)
	assert.NoError(t, err)
	assert.NoFileExists(t, filePath)
	assert.FileExists(t, filepath.Join(dst, "report.pdf"))
}

func TestAttemptMoveFile_IgnoresWrongSourceDir(t *testing.T) {
	src1 := t.TempDir()
	src2 := t.TempDir()
	dst := t.TempDir()

	// File is in src2, but category watches src1
	filePath := filepath.Join(src2, "file.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))

	cat := buildCategory("PDFs", src1, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})

	err := attemptMoveFile(m, filePath, false)
	assert.NoError(t, err)
	assert.FileExists(t, filePath)
}

// --- performInitialScan ---

func TestPerformInitialScan_AddsMatchingFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "a.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "b.txt"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})
	tracker := &fileTracker{files: make(map[string]time.Time)}

	performInitialScan(m, tracker)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	assert.Len(t, tracker.files, 1)
	assert.Contains(t, tracker.files, filepath.Join(src, "a.pdf"))
}

func TestPerformInitialScan_SkipsDisabledCategory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.pdf"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	cat.Enabled = boolPtr(false)
	m := newSilentMovelooper([]*models.Category{cat})
	tracker := &fileTracker{files: make(map[string]time.Time)}

	performInitialScan(m, tracker)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	assert.Empty(t, tracker.files)
}

func TestPerformInitialScan_IgnoresIgnoredFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "ignore_me.pdf"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "keep.pdf"), []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	cat.Source.Filter.Ignore = []string{"ignore_*"}
	m := newSilentMovelooper([]*models.Category{cat})
	tracker := &fileTracker{files: make(map[string]time.Time)}

	performInitialScan(m, tracker)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	assert.Len(t, tracker.files, 1)
	assert.Contains(t, tracker.files, filepath.Join(src, "keep.pdf"))
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

func TestProcessPendingFiles_MovesStableFile(t *testing.T) {
	m, dst, filePath, tracker := buildStaleTracker(t, "old.pdf")
	processPendingFiles(m, tracker, 5*time.Minute, false)
	assert.NoFileExists(t, filePath)
	assert.FileExists(t, filepath.Join(dst, "old.pdf"))
}

func TestProcessPendingFiles_SkipsFreshFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	filePath := filepath.Join(src, "fresh.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})
	tracker := &fileTracker{files: map[string]time.Time{
		filePath: time.Now(),
	}}

	// Threshold of 5 minutes, file was just written — should not move
	processPendingFiles(m, tracker, 5*time.Minute, false)

	assert.FileExists(t, filePath)
}

func TestProcessPendingFiles_RemovesDeletedFileFromTracker(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	ghostPath := filepath.Join(src, "ghost.pdf")

	cat := buildCategory("PDFs", src, dst, []string{"pdf"})
	m := newSilentMovelooper([]*models.Category{cat})
	tracker := &fileTracker{files: map[string]time.Time{
		ghostPath: time.Now().Add(-10 * time.Minute),
	}}

	processPendingFiles(m, tracker, 5*time.Minute, false)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	assert.NotContains(t, tracker.files, ghostPath)
}

func TestProcessPendingFiles_DryRunDoesNotMove(t *testing.T) {
	m, dst, filePath, tracker := buildStaleTracker(t, "stable.pdf")
	processPendingFiles(m, tracker, 5*time.Minute, true)
	// Dry-run: file stays in source
	assert.FileExists(t, filePath)
	assert.NoFileExists(t, filepath.Join(dst, "stable.pdf"))
}
