package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/lucasassuncao/bezel/component/progress"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/updater"
	"golang.org/x/term"
)

// downloadBarWidth is the bar's share of the line, beside its label and count.
const downloadBarWidth = 30

// newDownloadProgress draws self-update's download as a bar when stdout is a
// terminal; otherwise it returns nil and the updater prints a plain line.
func newDownloadProgress() updater.Progress {
	if !term.IsTerminal(int(os.Stdout.Fd())) { //#nosec G115 -- a stdout file descriptor always fits in an int
		return nil
	}
	return downloadBar(os.Stdout, theme.Resolve(theme.Default, theme.DarkTerminal()))
}

// downloadBar redraws the bar in place on w with \r, only when it changes:
// io.Copy reports every 32 KB, far more often than a tenth of a MB moves.
func downloadBar(w io.Writer, th theme.Resolved) updater.Progress {
	var last string
	return func(done, total int64) {
		if total <= 0 {
			return
		}
		if done >= total {
			if last != "" {
				fmt.Fprint(w, "\r\x1b[K") // erase the bar line; the next step prints its own
				last = ""
			}
			return
		}
		// per mille keeps the bar's int arithmetic safe on 32-bit builds
		bar := progress.Bar(int(done*1000/total), 1000, downloadBarWidth, th)
		count := th.Dim.Render(fmt.Sprintf("%.1f/%.1f MB", float64(done)/1e6, float64(total)/1e6))
		line := "downloading  " + bar + "  " + count
		if line != last {
			fmt.Fprint(w, "\r"+line)
			last = line
		}
	}
}
