package fileops

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
	"syscall"
)

// ErrFatalDestination marks a destination failure that will not resolve itself
// by moving on to the next file: the disk is full, or the process cannot write
// to the destination at all. Retrying every remaining file would produce the
// same error hundreds of times and bury the real cause, so MoveFiles stops the
// batch and reports it once.
var ErrFatalDestination = errors.New("destination is not writable")

// maxConsecutiveFatalErrors is how many back-to-back unrecoverable destination
// failures are tolerated before the batch gives up. It is not 1: a single
// ENOSPC can be one oversized file on a nearly-full disk, while smaller files
// after it still fit. Several in a row means the destination itself is done.
const maxConsecutiveFatalErrors = 5

// isFatalDestinationError reports whether err means the destination cannot be
// written to at all, as opposed to a problem with one particular file.
func isFatalDestinationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrPermission) {
		return true
	}
	if errors.Is(err, syscall.ENOSPC) {
		return true
	}
	if runtime.GOOS == "windows" {
		// ERROR_DISK_FULL (112) and ERROR_HANDLE_DISK_FULL (39) have no
		// syscall.ENOSPC equivalent on Windows.
		const windowsErrorDiskFull = syscall.Errno(112)
		const windowsErrorHandleDiskFull = syscall.Errno(39)
		if errors.Is(err, windowsErrorDiskFull) || errors.Is(err, windowsErrorHandleDiskFull) {
			return true
		}
	}
	return false
}

// sourceVanished reports whether err is a "not found" failure for a source file
// that is genuinely no longer on disk. This is the normal outcome when a second
// movelooper process (or the user) moved the file between the directory scan
// and the move itself — the file was not lost, it was already handled. Counting
// it as a failure would make a concurrent run exit non-zero for no reason.
func sourceVanished(src string, err error) bool {
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	_, statErr := os.Lstat(src)
	return os.IsNotExist(statErr)
}
