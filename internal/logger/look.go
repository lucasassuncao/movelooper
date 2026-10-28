package logger

import (
	"strconv"

	"charm.land/lipgloss/v2"
	"github.com/lucasassuncao/bezel/theme"
)

// Look paints the parts of a log message in the log's theme. The zero Look,
// and one for JSON, leaves them plain: a JSON line carries no colour.
type Look struct {
	th theme.Resolved
	on bool
}

// NewLook paints in th when on; the pretty log strips the colour itself
// wherever it writes without it.
func NewLook(th theme.Resolved, on bool) Look { return Look{th: th, on: on} }

// Category is a category's name in brackets, in the theme's accent.
func (l Look) Category(name string) string { return l.paint(l.th.Accent, "["+name+"]") }

// None is a word that says nothing was found, in the theme's danger colour.
func (l Look) None(s string) string { return l.paint(l.th.Danger, s) }

// Count is a number of files, in the theme's success colour.
func (l Look) Count(n int) string { return l.paint(l.th.Success, strconv.Itoa(n)) }

func (l Look) paint(st lipgloss.Style, s string) string {
	if !l.on {
		return s
	}
	return st.Render(s)
}
