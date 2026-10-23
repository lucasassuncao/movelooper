package fileops

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// renameNoReplace moves src to dst and fails, rather than replacing, when dst
// exists. Without MOVEFILE_COPY_ALLOWED a move across volumes fails with
// ERROR_NOT_SAME_DEVICE, which MoveFileCtx answers with a copy.
func renameNoReplace(src, dst string) error {
	from, err := windows.UTF16PtrFromString(longPath(src))
	if err == nil {
		var to *uint16
		if to, err = windows.UTF16PtrFromString(longPath(dst)); err == nil {
			err = windows.MoveFileEx(from, to, 0)
		}
	}
	if err != nil {
		return &os.LinkError{Op: "rename", Old: src, New: dst, Err: err}
	}
	return nil
}

// longPath gives a path past MAX_PATH the \?\ prefix, as package os does.
func longPath(p string) string {
	if len(p) < 248 || strings.HasPrefix(p, `\`) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return `\?\` + abs
}
