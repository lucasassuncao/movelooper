//go:build !windows

package hooks

import "os/exec"

// rawCmdLine does nothing outside Windows, where cmd does not run.
func rawCmdLine(*exec.Cmd, string) {}
