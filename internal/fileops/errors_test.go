package fileops

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFatalDestinationError(t *testing.T) {
	t.Parallel()

	assert.False(t, isFatalDestinationError(nil))
	assert.False(t, isFatalDestinationError(errors.New("some ordinary failure")))
	assert.False(t, isFatalDestinationError(fs.ErrNotExist), "a missing file is about one file, not the destination")

	assert.True(t, isFatalDestinationError(fs.ErrPermission))
	assert.True(t, isFatalDestinationError(syscall.ENOSPC))

	// Wrapped in the *os.PathError the standard library actually returns.
	wrapped := &os.PathError{Op: "open", Path: "/tmp/x", Err: syscall.ENOSPC}
	assert.True(t, isFatalDestinationError(wrapped))
}

func TestSourceVanished(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	present := filepath.Join(dir, "present.txt")
	require.NoError(t, os.WriteFile(present, []byte("hi"), 0o600))
	missing := filepath.Join(dir, "missing.txt")

	assert.False(t, sourceVanished(missing, nil), "no error means nothing vanished")
	assert.False(t, sourceVanished(missing, errors.New("unrelated")), "only not-exist errors qualify")
	assert.True(t, sourceVanished(missing, fs.ErrNotExist))

	// A not-exist error about the destination must not be mistaken for a
	// vanished source while the source is still sitting there.
	assert.False(t, sourceVanished(present, fs.ErrNotExist))
}
