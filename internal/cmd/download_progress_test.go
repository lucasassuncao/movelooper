package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/stretchr/testify/assert"
)

// go test captures stdout, so the bar is never drawn there.
func TestNewDownloadProgress_NilOutsideATerminal(t *testing.T) {
	t.Parallel()
	assert.Nil(t, newDownloadProgress(theme.Default))
}

func TestDownloadBar(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bar := downloadBar(&buf, theme.Resolve(theme.Default, true))

	bar(0, 0)
	assert.Empty(t, buf.String(), "an unknown size draws nothing")
	bar(25_000_000, 25_000_000)
	assert.Empty(t, buf.String(), "nothing drawn, nothing to erase")

	bar(13_100_000, 25_000_000)
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "\r"), "the bar redraws its own line")
	assert.Contains(t, out, "downloading")
	assert.Contains(t, out, "13.1/25.0 MB")

	buf.Reset()
	bar(13_120_000, 25_000_000)
	assert.Empty(t, buf.String(), "an unchanged line is not redrawn")

	bar(25_000_000, 25_000_000)
	assert.Equal(t, "\r\x1b[K", buf.String(), "a finished download erases the bar")
}
