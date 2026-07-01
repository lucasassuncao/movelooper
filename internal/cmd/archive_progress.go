package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// newArchiveProgress returns an archive progress callback that draws an inline
// bar to stdout, or nil when the session is not an interactive pretty-mode
// terminal (JSON logs, redirected/piped output, or a non-TTY), where a bar would
// corrupt the output or leak escape codes into a log file. The bar is written
// directly to os.Stdout, not through the logger, so file/"both" logs stay clean.
// The bar is transient: it redraws in place while archiving and erases itself on
// completion, leaving no misaligned residue between log lines — the "archived N
// files" log confirms completion.
func newArchiveProgress(m *models.Movelooper) func(done, total int) {
	if !interactiveTerminal(m) {
		return nil
	}
	bar := progress.New(progress.WithDefaultGradient(), progress.WithWidth(40))
	return func(done, total int) {
		if total <= 0 {
			return
		}
		if done >= total {
			fmt.Fprint(os.Stdout, "\r\x1b[K") // erase the bar line; the log line reports completion
			return
		}
		fmt.Fprintf(os.Stdout, "\rarchiving %d/%d %s", done, total, bar.ViewAs(float64(done)/float64(total)))
	}
}

// interactiveTerminal reports whether it is safe to draw a progress bar: the
// logger is the pretty (pterm) renderer — not the structured JSON logger — and
// stdout is a real terminal rather than a pipe or file.
func interactiveTerminal(m *models.Movelooper) bool {
	if _, pretty := m.Logger.(*pterm.Logger); !pretty {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd())) //#nosec G115 -- a stdout file descriptor always fits in an int
}
