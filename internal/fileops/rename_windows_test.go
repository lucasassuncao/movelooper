package fileops

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameNoReplace_Windows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	writeFile(t, src, []byte("x"))

	require.NoError(t, renameNoReplace(src, dst))
	assert.FileExists(t, dst)
	assert.NoFileExists(t, src)

	writeFile(t, src, []byte("y"))
	require.Error(t, renameNoReplace(src, dst), "a taken name is refused")
}

func TestLongPath(t *testing.T) {
	t.Parallel()
	short := `C:\short\path.txt`
	assert.Equal(t, short, longPath(short))
	long := `C:\` + strings.Repeat("d", 260) + `\f.txt`
	assert.Equal(t, `\?\`+long, longPath(long))
	unc := `\server\share\` + strings.Repeat("d", 260)
	assert.Equal(t, unc, longPath(unc), "already prefixed or UNC paths stay")
}
