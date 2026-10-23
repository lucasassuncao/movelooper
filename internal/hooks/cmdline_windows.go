package hooks

import (
	"os/exec"
	"syscall"
)

// rawCmdLine hands cmd the command as written: Go would escape its inner
// quotes with backslashes, which cmd does not understand.
func rawCmdLine(c *exec.Cmd, command string) {
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(c.Path) + " /C " + command}
}
