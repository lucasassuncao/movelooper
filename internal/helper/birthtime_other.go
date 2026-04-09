//go:build !windows && !darwin

package helper

import (
	"os"
	"time"
)

// getBirthTime falls back to modification time on platforms that do not
// expose a file creation timestamp (e.g. Linux).
func getBirthTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
