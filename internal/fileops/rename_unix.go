//go:build !windows

package fileops

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// renameNoReplace moves src to dst and fails, rather than replacing, when dst
// exists: a hard link refuses a taken name, and the old name is then dropped.
func renameNoReplace(src, dst string) error {
	err := os.Link(src, dst)
	switch {
	case err == nil:
		if rmErr := os.Remove(src); rmErr != nil {
			_ = os.Remove(dst)
			return &os.LinkError{Op: "rename", Old: src, New: dst, Err: rmErr}
		}
		return nil
	case errors.Is(err, fs.ErrExist), errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.EXDEV):
		return err
	}
	// A filesystem without hard links: check the name, then rename.
	if _, statErr := os.Lstat(dst); statErr == nil {
		return &os.LinkError{Op: "rename", Old: src, New: dst, Err: fs.ErrExist}
	}
	return os.Rename(src, dst)
}
