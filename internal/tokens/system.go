package tokens

import (
	"os"
	"os/user"
	"runtime"
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
			systemUsername = u.Username
		} else {
			systemUsername = "unknown"
		}
	})
}
