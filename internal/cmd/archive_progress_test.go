package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
)

// A bar drawn into JSON logs or a pipe would corrupt them, so both get nil.
func TestNewArchiveProgress_NilOutsideAnInteractiveTerminal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	structured := &models.Movelooper{Logger: logger.NewSlog(&buf, "info", false)}
	assert.False(t, isInteractiveTerminal(structured))
	assert.Nil(t, newArchiveProgress(structured))

	// go test captures stdout, so a pretty logger still is not on a terminal
	pretty := &models.Movelooper{Logger: pterm.DefaultLogger.WithWriter(&buf)}
	assert.False(t, isInteractiveTerminal(pretty))
	assert.Nil(t, newArchiveProgress(pretty))
}

func TestArchiveBar(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bar := archiveBar(&buf, theme.Resolve(theme.ThemeDefault, theme.DarkTerminal()))

	bar(0, 0)
	assert.Empty(t, buf.String(), "an empty batch draws nothing")

	bar(3, 10)
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "\r"), "the bar redraws its own line")
	assert.Contains(t, out, "archiving")
	assert.Contains(t, out, "3/10")

	buf.Reset()
	bar(10, 10)
	assert.Equal(t, "\r\x1b[K", buf.String(), "a finished batch erases the bar")
}
