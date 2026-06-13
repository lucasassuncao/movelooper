package tokens

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetBirthTime tests that getBirthTime returns a recent, non-zero time for a newly created file.
func TestGetBirthTime(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)

	got := getBirthTime(info)

	assert.False(t, got.IsZero(), "birth time should not be zero")
	assert.WithinDuration(t, time.Now(), got, 10*time.Second, "birth time should be recent")
}

// TestGetBirthTime_ModifiedMtime tests that getBirthTime returns a non-zero time even when the file's mtime has been modified.
func TestGetBirthTime_ModifiedMtime(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	modTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(path, modTime, modTime))

	info, err := os.Stat(path)
	require.NoError(t, err)

	got := getBirthTime(info)

	assert.False(t, got.IsZero(), "birth time should not be zero even with modified mtime")
}
