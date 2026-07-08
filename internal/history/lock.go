package history

import "os"

// fileLock is an OS-level advisory lock on a sidecar file, used to serialize
// writes to the history file across movelooper processes (e.g. a watch daemon
// and a one-shot run writing at the same time). The in-process h.mu mutex only
// protects goroutines within one process; this closes the remaining gap.
// lockFile and unlockFile are implemented per-OS in lock_unix.go / lock_windows.go.
type fileLock struct {
	f *os.File
}

// acquireFileLock opens (creating if needed) the lock file at path and blocks
// until an exclusive lock is held.
func acquireFileLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //#nosec G304 -- path is the history file's path plus ".lock", set by the application at startup
	if err != nil {
		return nil, err
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &fileLock{f: f}, nil
}

// release unlocks and closes the underlying file.
func (l *fileLock) release() error {
	unlockErr := unlockFile(l.f)
	closeErr := l.f.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
