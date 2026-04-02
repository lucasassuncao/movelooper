// Package terminal provides utilities for interacting with the terminal.
package terminal

import (
	"os"
	"os/exec"
	"runtime"
)

// ClearScreen clears the terminal screen based on the operating system.
func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
