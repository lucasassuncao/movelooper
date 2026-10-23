package cmd

import (
	"os"
	"path/filepath"
	"strings"
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
		assert.Empty(t, strictDirViolations(yaml, ""))
	})

	t.Run("missing tilde path is still flagged", func(t *testing.T) {
		yaml := []byte("categories:\n  - source:\n      path: ~/does-not-exist\n    destination:\n      path: ~/dst\n")
		v := strictDirViolations(yaml, "")
		require.Len(t, v, 1)
		assert.Contains(t, v[0].Path, "source.path")
	})
}

// writeConfig writes the main config and the files it imports into one folder.
func writeConfig(t *testing.T, main string, imports map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range imports {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
	}
	path := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(path, []byte(main), 0o600))
	return path
}

// Validate used to read the main file alone: an imported category broken
// enough to stop the run passed with "ok".
func TestCollectViolations_ChecksImportedFiles(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "import:\n  - ./more.yaml\ncategories: []\n", map[string]string{
		"more.yaml": "categories:\n  - name: imported\n    enabled: true\n    source:\n      path: /in\n      extensions: [txt]\n    destination:\n      path: /out\n      conflict-strategy: banana\n",
	})

	violations, err := collectViolations(path, false)
	require.NoError(t, err)

	var found bool
	for _, v := range violations {
		if strings.HasPrefix(v.Path, "more.yaml") && strings.Contains(v.Path, "conflict-strategy") {
			found = true
		}
	}
	assert.True(t, found, "the imported file's violation, named by its file: %v", violations)
}

func TestCollectViolations_ReportsAMissingImport(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "import:\n  - ./gone.yaml\ncategories: []\n", nil)

	violations, err := collectViolations(path, false)
	require.NoError(t, err)
	paths := make([]string, len(violations))
	for i, v := range violations {
		paths[i] = v.Path
	}
	assert.Contains(t, paths, "import[0]")
}
