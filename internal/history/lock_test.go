package history

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquireFileLock_ReleaseAllowsReacquire(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := acquireFileLock(path)
	require.NoError(t, err)
	require.NoError(t, lock.release())

	lock2, err := acquireFileLock(path)
	require.NoError(t, err)
	require.NoError(t, lock2.release())
}

// TestAcquireFileLock_BlocksSecondHolder proves the lock is actually
// exclusive: a second acquire on the same path blocks until the first is
// released, rather than succeeding immediately.
func TestAcquireFileLock_BlocksSecondHolder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := acquireFileLock(path)
	require.NoError(t, err)

	acquired := make(chan struct{})
	go func() {
		lock2, err := acquireFileLock(path)
		require.NoError(t, err)
		close(acquired)
		_ = lock2.release()
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire succeeded while the first lock was still held")
	case <-time.After(150 * time.Millisecond):
		// expected: still blocked
	}

	require.NoError(t, lock.release())

	select {
	case <-acquired:
		// expected: unblocked after release
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not unblock after release")
	}
}
