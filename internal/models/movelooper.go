package models

import (
	"io"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/pterm/pterm"
)

// Movelooper holds the app dependencies and runtime state.
// Viper is intentionally absent: it is used only during initialisation
// in preRunHandler and discarded afterwards.
type Movelooper struct {
	Logger     *pterm.Logger
	Config     Configuration
	Categories []*Category
	History    *history.History
	LogCloser  io.Closer // non-nil when logging to a file; closed on exit
}
