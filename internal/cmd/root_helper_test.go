package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogExtensionResult_PlainWhenColorDisabled verifies that the scan summary
// message carries the category and count, and that with color disabled (as the
// JSON logger does) it contains no ANSI escape codes, keeping structured logs clean.
func TestLogExtensionResult_PlainWhenColorDisabled(t *testing.T) {
	// Not parallel: toggles pterm's global color state.
	pterm.DisableColor()
	t.Cleanup(func() { pterm.EnableColor() })

	var buf bytes.Buffer
	m := &models.Movelooper{Logger: logger.NewSlog(&buf, "info", false)}

	logExtensionResult(m, nil, "images", "jpg", false)

	dir := t.TempDir()
	for _, name := range []string{"a.jpg", "b.jpg"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	logExtensionResult(m, entries, "images", "jpg", false)

	out := buf.String()
	assert.NotContains(t, out, "\x1b", "JSON output must not contain ANSI color codes")
	assert.Contains(t, out, "[images]")
	assert.Contains(t, out, "No .jpg files found")
	assert.Contains(t, out, "2 .jpg files to move")
}

// TestFileNoun covers the scan-summary subject for both real extensions and the
// "all" sentinel, where ".all" is meaningless and the noun must be singular/plural.
func TestFileNoun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ext   string
		count int
		want  string
	}{
		{"jpg", 0, ".jpg files"},
		{"jpg", 1, ".jpg file"},
		{"jpg", 5, ".jpg files"},
		{"all", 0, "files"},
		{"all", 1, "file"},
		{"all", 2, "files"},
		{"ALL", 1, "file"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, fileNoun(c.ext, c.count), "fileNoun(%q, %d)", c.ext, c.count)
	}
}
