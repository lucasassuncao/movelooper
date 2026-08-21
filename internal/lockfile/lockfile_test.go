package lockfile

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquire_ReleaseAllowsReacquire(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	require.NoError(t, err)
	require.NoError(t, lock.Release())

	lock2, err := Acquire(path)
	require.NoError(t, err)
	require.NoError(t, lock2.Release())
}

// TestAcquire_BlocksSecondHolder proves the lock is actually exclusive: a
// second acquire on the same path blocks until the first is released, rather
// than succeeding immediately.
func TestAcquire_BlocksSecondHolder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(path)
	require.NoError(t, err)

	acquired := make(chan struct{})
	go func() {
		lock2, err := Acquire(path)
		require.NoError(t, err)
		close(acquired)
		_ = lock2.Release()
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire succeeded while the first lock was still held")
	case <-time.After(150 * time.Millisecond):
		// expected: still blocked
	}

	require.NoError(t, lock.Release())

	select {
	case <-acquired:
		// expected: unblocked after release
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not unblock after release")
	}
}

func TestTryAcquire_ReportsHeldLockImmediately(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := TryAcquire(path)
	require.NoError(t, err)
	defer func() { _ = lock.Release() }()

	start := time.Now()
	second, err := TryAcquire(path)
	require.ErrorIs(t, err, ErrLocked)
	require.Nil(t, second)
	require.Less(t, time.Since(start), time.Second, "TryAcquire must not block")
}

func TestTryAcquire_SucceedsAfterRelease(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock")

	lock, err := TryAcquire(path)
	require.NoError(t, err)
	require.NoError(t, lock.Release())

	second, err := TryAcquire(path)
	require.NoError(t, err)
	require.NoError(t, second.Release())
}

// TestTryAcquire_CreatesMissingParentDirectory covers the watch lock, whose
// directory (~/.movelooper) may not exist on a first run.
func TestTryAcquire_CreatesMissingParentDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "dir", "test.lock")

	lock, err := TryAcquire(path)
	require.NoError(t, err)
	require.NoError(t, lock.Release())
	require.FileExists(t, path)
}

func TestErrLocked_IsMatchable(t *testing.T) {
	t.Parallel()
	require.True(t, errors.Is(ErrLocked, ErrLocked))
}
