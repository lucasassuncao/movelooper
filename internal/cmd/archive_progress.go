package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/lucasassuncao/bezel/component/progress"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"golang.org/x/term"
)

// newArchiveProgress draws a transient inline bar to stdout when the session
// is an interactive pretty terminal; otherwise it returns nil to avoid corrupting
// logs or emitting escape codes to redirected output.
func newArchiveProgress(m *models.Movelooper) func(done, total int) {
	if !isInteractiveTerminal(m) {
		return nil
	}
	return archiveBar(os.Stdout, theme.Resolve(m.Theme, theme.DarkTerminal()))
}

// archiveBar redraws the bar in place on w with \r.
func archiveBar(w io.Writer, th theme.Resolved) func(done, total int) {
	return func(done, total int) {
		if total <= 0 {
			return
		}
		if done >= total {
			fmt.Fprint(w, "\r\x1b[K") // erase the bar line; the log line reports completion
			return
		}
		fmt.Fprint(w, "\r"+progress.Line("archiving", done, total, 56, th))
	}
}

// isInteractiveTerminal reports whether it is safe to draw a progress bar: the
// logger is the pretty renderer — not the structured JSON logger — and
// stdout is a real terminal rather than a pipe or file.
func isInteractiveTerminal(m *models.Movelooper) bool {
	if !logger.Pretty(m.Logger) {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd())) //#nosec G115 -- a stdout file descriptor always fits in an int
}
