package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCategoriesWithHooks verifies that only categories defining a before or
// after hook are reported, so watch mode warns about exactly those.
func TestCategoriesWithHooks(t *testing.T) {
	t.Parallel()

	cats := []*models.Category{
		{Name: "no-hooks"},
		{Name: "empty-hooks", Hooks: &models.CategoryHooks{}},
		{Name: "with-before", Hooks: &models.CategoryHooks{Before: &models.CategoryHook{}}},
		{Name: "with-after", Hooks: &models.CategoryHooks{After: &models.CategoryHook{}}},
	}

	assert.Equal(t, []string{"with-before", "with-after"}, categoriesWithHooks(cats))
}

// TestResolveDestDir covers the watch-mode destination resolution: the plain
// destination, the organize-by subdir, and the fallback when the file is gone.
func TestResolveDestDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jpg := filepath.Join(dir, "photo.jpg")
	require.NoError(t, os.WriteFile(jpg, []byte("x"), 0o644))

	t.Run("no organize-by returns destination path", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Destination: models.CategoryDestination{Path: "/dest"}}
		assert.Equal(t, "/dest", resolveDestDir(cat, jpg))
	})
	t.Run("organize-by appends resolved subdir", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Destination: models.CategoryDestination{Path: "/dest", OrganizeBy: "{ext}"}}
		assert.Equal(t, filepath.Join("/dest", "jpg"), resolveDestDir(cat, jpg))
	})
	t.Run("missing file falls back to destination path", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Destination: models.CategoryDestination{Path: "/dest", OrganizeBy: "{ext}"}}
		assert.Equal(t, "/dest", resolveDestDir(cat, filepath.Join(dir, "gone.jpg")))
	})
}

// TestMatchesExtensionAndFilters covers extension matching plus the name filter
// applied by the watch path before a file is moved.
func TestMatchesExtensionAndFilters(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jpg := filepath.Join(dir, "photo.jpg")
	require.NoError(t, os.WriteFile(jpg, []byte("x"), 0o644))

	t.Run("matching extension passes", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Source: models.CategorySource{Extensions: []string{"jpg"}}}
		assert.True(t, matchesExtensionAndFilters(cat, "photo.jpg", jpg))
	})
	t.Run("non-matching extension fails", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Source: models.CategorySource{Extensions: []string{"png"}}}
		assert.False(t, matchesExtensionAndFilters(cat, "photo.jpg", jpg))
	})
	t.Run("not filter excludes the file", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Source: models.CategorySource{
			Extensions: []string{"jpg"},
			Filter:     models.CategoryFilter{Not: []models.CategoryFilter{{Match: &models.MatchFilter{Glob: "photo*"}}}},
		}}
		assert.False(t, matchesExtensionAndFilters(cat, "photo.jpg", jpg))
	})
	t.Run("missing file fails", func(t *testing.T) {
		t.Parallel()
		cat := &models.Category{Source: models.CategorySource{Extensions: []string{"jpg"}}}
		assert.False(t, matchesExtensionAndFilters(cat, "gone.jpg", filepath.Join(dir, "gone.jpg")))
	})
	t.Run("directory with a matching extension is rejected", func(t *testing.T) {
		t.Parallel()
		subdir := filepath.Join(dir, "vacation.jpg")
		require.NoError(t, os.Mkdir(subdir, 0o750))
		cat := &models.Category{Source: models.CategorySource{Extensions: []string{"all"}}}
		assert.False(t, matchesExtensionAndFilters(cat, "vacation.jpg", subdir))
	})
	t.Run("symlink with a matching extension is rejected", func(t *testing.T) {
		t.Parallel()
		link := filepath.Join(dir, "link.jpg")
		if err := os.Symlink(jpg, link); err != nil {
			t.Skipf("symlinks not supported on this platform/user: %v", err)
		}
		cat := &models.Category{Source: models.CategorySource{Extensions: []string{"jpg"}}}
		assert.False(t, matchesExtensionAndFilters(cat, "link.jpg", link))
	})
}

// TestAcquireLockAt covers the watch lock: it is exclusive while held, it is
// released cleanly, and a lock file left behind by a process that died is
// acquired without complaint — the OS drops the lock with the process, so
// there is no stale state to detect.
func TestAcquireLockAt(t *testing.T) {
	t.Parallel()

	t.Run("creates the lock file and releases it", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "test.lock")
		release, err := acquireLockAt(path)
		require.NoError(t, err)
		assert.FileExists(t, path)
		release()

		// The file is intentionally left on disk: it carries no state, and
		// removing it would race with another process holding it open.
		assert.FileExists(t, path)
	})

	t.Run("rejects a second holder while the lock is held", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "test.lock")
		release, err := acquireLockAt(path)
		require.NoError(t, err)
		defer release()

		_, err = acquireLockAt(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("allows a new holder after release", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "test.lock")
		release, err := acquireLockAt(path)
		require.NoError(t, err)
		release()

		release2, err := acquireLockAt(path)
		require.NoError(t, err)
		release2()
	})

	// A leftover file from a killed watcher used to strand the lock forever on
	// Windows, where the liveness probe always answered "still running".
	t.Run("acquires a lock file left behind by a dead process", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "test.lock")
		require.NoError(t, os.WriteFile(path, []byte("4084\n"), 0o600))

		release, err := acquireLockAt(path)
		require.NoError(t, err)
		defer release()
	})
}
