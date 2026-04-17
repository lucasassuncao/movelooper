package helper

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
)

// RunHook executes all commands in hook.Run sequentially, injecting env into
// each process environment. Returns nil when hook is nil.
// On command failure:
//   - "abort": stops execution and returns the error.
//   - "warn":  logs the error and continues to the next command.
func RunHook(hook *models.CategoryHook, logger *pterm.Logger, env map[string]string) error {
	if hook == nil {
		return nil
	}

	shell, shellArgs := defaultShell(hook.Shell)
	combined := buildEnv(env)

	for _, command := range hook.Run {
		args := append(shellArgs, command)  //nolint:gocritic
		cmd := exec.Command(shell, args...) //#nosec G204 -- shell and command are user-defined config values
		cmd.Env = combined
		setSysProcAttr(cmd)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			msg := fmt.Sprintf("hook command failed: %s", command)
			if hook.OnFailure == "abort" {
				return fmt.Errorf("%s: %w", msg, err)
			}
			logger.Warn(msg, logger.Args("error", err.Error()))
		}
	}
	return nil
}

// defaultShell returns the shell executable and the flag that introduces a
// command string ("-c" on Unix, "/C" on Windows cmd, "-Command" for PowerShell).
// If override is non-empty it is used as-is; otherwise the system default is detected.
func defaultShell(override string) (shell string, args []string) {
	if override != "" {
		if runtime.GOOS == "windows" {
			switch override {
			case "powershell", "pwsh":
				return override, []string{"-NonInteractive", "-NoProfile", "-Command"}
			default:
				return override, []string{"/C"}
			}
		}
		return override, []string{"-c"}
	}

	if runtime.GOOS == "windows" {
		if sh := os.Getenv("SHELL"); sh != "" {
			return sh, []string{"-c"}
		}
		return "cmd", []string{"/C"}
	}

	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-c"}
	}
	return "sh", []string{"-c"}
}

// buildEnv merges the current process environment with the provided extra vars.
// Extra vars take precedence via appending (OS uses last occurrence on most platforms).
func buildEnv(extra map[string]string) []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+len(extra))
	env = append(env, base...)
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}
