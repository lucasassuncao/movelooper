//go:build darwin

package helper

import (
	"os"
	"syscall"
	"time"
)

// getBirthTime returns the file creation time on macOS.
func getBirthTime(info os.FileInfo) time.Time {
	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.ModTime()
	}
	return time.Unix(sys.Birthtimespec.Sec, sys.Birthtimespec.Nsec)
}
