package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpandTilde covers the leading-tilde expansion and the cases that must be
// left untouched (absolute, relative, empty, and "~username").
func TestExpandTilde(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare tilde", "~", home},
		{"tilde slash", "~/Downloads", filepath.Join(home, "Downloads")},
		{"absolute unchanged", filepath.Join(home, "abs"), filepath.Join(home, "abs")},
		{"relative unchanged", "relative/dir", "relative/dir"},
		{"empty unchanged", "", ""},
		{"named home not expanded", "~user/x", "~user/x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, ExpandTilde(c.in))
		})
	}
}
