// Package logger builds the app's *slog.Logger: bezel's tree for people, or
// JSON lines for machines.
package logger

import (
	"io"
	"log/slog"

	bezellog "github.com/lucasassuncao/bezel/component/logger"
	"github.com/lucasassuncao/bezel/theme"
)

// slog has no Trace or Fatal levels; these sit below Debug and above Error so
// the configured trace/fatal vocabulary survives both renderers.
const (
	LevelTrace = bezellog.LevelTrace
	levelFatal = slog.Level(12)
)

// NewJSON logs JSON lines. addSource adds each line's caller, from the hidden
// --show-caller flag.
func NewJSON(w io.Writer, level string, addSource bool) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       Level(level),
		AddSource:   addSource,
		ReplaceAttr: renameCustomLevels,
	}))
}

// NewPretty logs each message with its arguments as a tree under it, in th.
// color is "always", "never" or "auto" (colour on a terminal); width 0 wraps
// at the terminal's width.
func NewPretty(w io.Writer, th theme.Resolved, level string, addSource bool, color string, width int) *slog.Logger {
	h := bezellog.New(w, th).
		WithLevel(Level(level)).
		WithCaller(addSource).
		WithWidth(width)
	switch color {
	case "always":
		h = h.WithColor(true)
	case "never":
		h = h.WithColor(false)
	}
	return slog.New(h)
}

// Discard drops every line, for tests and code paths that log nothing.
func Discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

// Pretty reports whether l prints for people rather than as JSON.
func Pretty(l *slog.Logger) bool {
	_, ok := l.Handler().(bezellog.Handler)
	return ok
}

// Level maps the configured level name to its slog level; unknown is info.
func Level(level string) slog.Level {
	switch level {
	case "trace":
		return LevelTrace
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "fatal":
		return levelFatal
	default:
		return slog.LevelInfo
	}
}

// renameCustomLevels names trace and fatal in JSON instead of slog's
// "DEBUG-4" and "ERROR+4".
func renameCustomLevels(groups []string, a slog.Attr) slog.Attr {
	if lvl, ok := a.Value.Any().(slog.Level); ok && a.Key == slog.LevelKey && lvl == levelFatal {
		a.Value = slog.StringValue("FATAL")
		return a
	}
	return bezellog.ReplaceAttr(groups, a)
}
