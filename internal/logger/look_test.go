package logger

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/stretchr/testify/assert"
)

func TestLook_PaintsInTheTheme(t *testing.T) {
	t.Parallel()
	l := NewLook(theme.Resolve(theme.Default(), true), true)
	for _, s := range []string{l.Category("images"), l.None("No"), l.Count(3)} {
		assert.Contains(t, s, "\x1b[")
	}
	assert.Equal(t, "[images]", ansi.Strip(l.Category("images")))
	assert.Equal(t, "3", ansi.Strip(l.Count(3)))
}

func TestLook_StaysPlainWhenOff(t *testing.T) {
	t.Parallel()
	var zero Look
	assert.Equal(t, "[images]", zero.Category("images"))
	assert.Equal(t, "No", zero.None("No"))
	assert.Equal(t, "12", NewLook(theme.Resolve(theme.Default(), true), false).Count(12))
}
