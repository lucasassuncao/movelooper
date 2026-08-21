//go:build !windows

package lockfile

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// lockFile takes an exclusive flock on f. When nonBlocking is set the call
// returns EWOULDBLOCK instead of waiting for the current holder.
func lockFile(f *os.File, nonBlocking bool) error {
	how := unix.LOCK_EX
	if nonBlocking {
		how |= unix.LOCK_NB
	}
	return unix.Flock(int(f.Fd()), how) //#nosec G115 -- a file descriptor always fits in an int
}

// unlockFile releases the lock held by lockFile.
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN) //#nosec G115 -- a file descriptor always fits in an int
}

// isWouldBlock reports whether err means "another process holds the lock".
func isWouldBlock(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}
