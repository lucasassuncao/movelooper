//go:build !windows

package history

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile blocks until an exclusive advisory lock on f is held.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX) //#nosec G115 -- a file descriptor always fits in an int
}

// unlockFile releases the lock held by lockFile.
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN) //#nosec G115 -- a file descriptor always fits in an int
}
