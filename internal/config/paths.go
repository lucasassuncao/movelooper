package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandTilde expands a leading "~" or "~/" (and "~\" on Windows) in path to the
// user's home directory. Any other value — including a bare "~username" — is
// returned unchanged, as is path when the home directory cannot be resolved.
func ExpandTilde(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
