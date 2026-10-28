package models

import (
	"io"
	"log/slog"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/logger"
)

// Movelooper holds the app dependencies and runtime state.
// Viper is intentionally absent: it is used only during initialisation
// in preRunHandler and discarded afterwards.
type Movelooper struct {
	Logger     *slog.Logger
	Look       logger.Look // colours the parts of a log message
	Theme      theme.Theme // from --theme or configuration.logging.theme
	Config     Configuration
	Categories []*Category
	History    *history.History
	LogCloser  io.Closer // non-nil when logging to a file; closed on exit
}
