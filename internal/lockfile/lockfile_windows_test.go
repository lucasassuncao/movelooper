package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

// Two handles on the same file conflict even inside one process.
func TestLockFile_SecondHandleWouldBlock(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	first, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer first.Close()
	second, err := os.OpenFile(path, os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer second.Close()

	require.NoError(t, lockFile(first, true))
	err = lockFile(second, true)
	require.Error(t, err)
	assert.True(t, isWouldBlock(err))

	require.NoError(t, unlockFile(first))
	require.NoError(t, lockFile(second, true), "free once the first handle unlocks")
	require.NoError(t, unlockFile(second))
}

func TestIsWouldBlock(t *testing.T) {
	t.Parallel()
	assert.True(t, isWouldBlock(windows.ERROR_LOCK_VIOLATION))
	assert.False(t, isWouldBlock(errors.New("other")))
	assert.False(t, isWouldBlock(nil))
}
