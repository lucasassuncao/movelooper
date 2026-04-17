//go:build windows

package helper

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd) {}
