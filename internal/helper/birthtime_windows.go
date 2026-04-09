//go:build windows

package helper

import (
	"os"
	"syscall"
	"time"
)

// getBirthTime returns the file creation time on Windows.
func getBirthTime(info os.FileInfo) time.Time {
	sys, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return info.ModTime()
	}
	return time.Unix(0, sys.CreationTime.Nanoseconds())
}
