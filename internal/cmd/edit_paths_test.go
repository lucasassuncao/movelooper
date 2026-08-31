package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveEditPaths_firstRunTargetsHomeConfig guards the first-run path: with
// no config anywhere, "movelooper edit" must point at the first location
// ResolveConfigPath searches, so the file it creates is the one every later run
// finds.
func TestResolveEditPaths_firstRunTargetsHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	loadPath, savePath, err := resolveEditPaths("", "")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, ".movelooper", "conf", "movelooper.yaml"), loadPath)
	assert.Empty(t, savePath)
}

// TestEnsureConfigDir_createsMissingParent covers the save that follows: yedit
// writes its temp file beside the target, so the parent must exist before the
// editor opens.
func TestEnsureConfigDir_createsMissingParent(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), ".movelooper", "conf", "movelooper.yaml")

	require.NoError(t, ensureConfigDir(target))

	info, err := os.Stat(filepath.Dir(target))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestEnsureConfigDir_existingDirIsNoop keeps a repeated run from failing on a
// directory that is already there.
func TestEnsureConfigDir_existingDirIsNoop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, ensureConfigDir(filepath.Join(dir, "movelooper.yaml")))
}
