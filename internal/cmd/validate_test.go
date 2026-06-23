package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStrictDirViolations_ExpandsTilde verifies that --strict resolves a leading
// "~" in source/destination paths before checking existence, so a real config
// using "~/Downloads" is not falsely flagged as a missing directory.
func TestStrictDirViolations_ExpandsTilde(t *testing.T) {
	// t.Setenv forbids t.Parallel on this test.
	home := t.TempDir()
	t.Setenv("HOME", home)        // Unix home lookup
	t.Setenv("USERPROFILE", home) // Windows home lookup
	require.NoError(t, os.MkdirAll(filepath.Join(home, "src"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "dst"), 0o750))

	t.Run("existing tilde paths produce no violations", func(t *testing.T) {
		yaml := []byte("categories:\n  - source:\n      path: ~/src\n    destination:\n      path: ~/dst\n")
		assert.Empty(t, strictDirViolations(yaml))
	})

	t.Run("missing tilde path is still flagged", func(t *testing.T) {
		yaml := []byte("categories:\n  - source:\n      path: ~/does-not-exist\n    destination:\n      path: ~/dst\n")
		v := strictDirViolations(yaml)
		require.Len(t, v, 1)
		assert.Contains(t, v[0].Path, "source.path")
	})
}
