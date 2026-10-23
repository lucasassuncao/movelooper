//go:build !windows

package fileops

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameNoReplace_Unix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	writeFile(t, src, []byte("x"))

	require.NoError(t, renameNoReplace(src, dst))
	assert.FileExists(t, dst)
	assert.NoFileExists(t, src)

	writeFile(t, src, []byte("y"))
	err := renameNoReplace(src, dst)
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrExist))
	assert.FileExists(t, src, "the refused source stays")
}
