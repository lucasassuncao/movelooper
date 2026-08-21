// Package lockfile provides exclusive, OS-level advisory locks on a sidecar
// file, used to serialize work across movelooper processes.
//
// The lock is held by the operating system for as long as the owning process
// holds the file open. It is released automatically when the process exits,
// including on a crash or a kill that runs no deferred code, so a lock can
// never be left stranded. That is the whole reason this package exists: an
// earlier PID-file scheme had to guess whether the recorded process was still
// alive, and guessed wrong on Windows.
//
// The lock file itself is deliberately never deleted. It carries no state — it
// is only a handle for the OS lock — and removing it would race with another
// process that has it open.
package lockfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrLocked is returned by TryAcquire when another process holds the lock.
var ErrLocked = errors.New("lock is held by another process")

// Lock is an acquired exclusive lock. Release must be called to free it,
// although process exit also releases it.
type Lock struct {
	f *os.File
}

// Acquire opens (creating if needed) the lock file at path and blocks until an
// exclusive lock is held. Parent directories are created as needed.
func Acquire(path string) (*Lock, error) {
	return acquire(path, false)
}

// TryAcquire behaves like Acquire but never blocks: when another process
// already holds the lock it returns ErrLocked immediately.
func TryAcquire(path string) (*Lock, error) {
	return acquire(path, true)
}

func acquire(path string, nonBlocking bool) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create lock directory for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //#nosec G304 -- path is built by the application, not taken from user input
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}
	if err := lockFile(f, nonBlocking); err != nil {
		_ = f.Close()
		if nonBlocking && isWouldBlock(err) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return &Lock{f: f}, nil
}

// Release unlocks and closes the underlying file. The lock file is left on disk
// on purpose; see the package comment.
func (l *Lock) Release() error {
	unlockErr := unlockFile(l.f)
	closeErr := l.f.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
