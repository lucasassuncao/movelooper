package tokens

import (
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
)

var (
	systemOnce     sync.Once
	systemHostname string
	systemUsername string
	systemOS       string
)

func initSystemContext() {
	systemOnce.Do(func() {
		systemOS = runtime.GOOS
		if h, err := os.Hostname(); err == nil {
			systemHostname = h
		} else {
			systemHostname = "unknown"
		}
		if u, err := user.Current(); err == nil {
			systemUsername = stripDomain(u.Username)
		} else {
			systemUsername = "unknown"
		}
	})
}

// stripDomain removes a leading "DOMAIN\" (or "domain/") qualifier that
// user.Current().Username carries on Windows, so {username} never introduces a
// path separator into an organize-by subdirectory.
func stripDomain(username string) string {
	if i := strings.LastIndexAny(username, `\/`); i >= 0 {
		return username[i+1:]
	}
	return username
}
