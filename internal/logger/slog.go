package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/pterm/pterm"
)

// slog has no Trace or Fatal levels; map them to values below Debug and above
// Error so the existing trace/fatal vocabulary survives the JSON renderer.
const (
	levelTrace = slog.Level(-8)
	levelFatal = slog.Level(12)
)

// slogLogger adapts an *slog.Logger to the Logger interface so that the JSON
// format can reuse every call site unchanged. Arguments arrive as
// []pterm.LoggerArgument (the shared transport type) and are converted to
// slog attributes on the way out.
type slogLogger struct {
	l *slog.Logger
}

// NewSlog builds a Logger that emits structured JSON via slog. addSource mirrors
// the pretty logger's show-caller setting.
func NewSlog(w io.Writer, level string, addSource bool) Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       slogLevel(level),
		AddSource:   addSource,
		ReplaceAttr: renameCustomLevels,
	})
	return &slogLogger{l: slog.New(h)}
}

func (s *slogLogger) Trace(msg string, args ...[]pterm.LoggerArgument) { s.log(levelTrace, msg, args) }
func (s *slogLogger) Debug(msg string, args ...[]pterm.LoggerArgument) {
	s.log(slog.LevelDebug, msg, args)
}
func (s *slogLogger) Info(msg string, args ...[]pterm.LoggerArgument) {
	s.log(slog.LevelInfo, msg, args)
}
func (s *slogLogger) Warn(msg string, args ...[]pterm.LoggerArgument) {
	s.log(slog.LevelWarn, msg, args)
}
func (s *slogLogger) Error(msg string, args ...[]pterm.LoggerArgument) {
	s.log(slog.LevelError, msg, args)
}

// Args packs alternating key/value pairs into the shared LoggerArgument slice,
// matching pterm's Args contract.
func (s *slogLogger) Args(args ...interface{}) []pterm.LoggerArgument {
	out := make([]pterm.LoggerArgument, 0, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		out = append(out, pterm.LoggerArgument{Key: fmt.Sprint(args[i]), Value: args[i+1]})
	}
	return out
}

func (s *slogLogger) log(level slog.Level, msg string, groups [][]pterm.LoggerArgument) {
	var attrs []slog.Attr
	for _, group := range groups {
		for _, a := range group {
			attrs = append(attrs, slog.Any(a.Key, a.Value))
		}
	}
	s.l.LogAttrs(context.Background(), level, msg, attrs...)
}

// slogLevel maps the configured level string to a slog level threshold.
func slogLevel(level string) slog.Level {
	switch level {
	case "trace":
		return levelTrace
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
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

// renameCustomLevels gives the out-of-band trace/fatal levels readable labels
// instead of slog's "DEBUG-4"/"ERROR+4" defaults.
func renameCustomLevels(_ []string, a slog.Attr) slog.Attr {
	if a.Key != slog.LevelKey {
		return a
	}
	if lvl, ok := a.Value.Any().(slog.Level); ok {
		switch lvl {
		case levelTrace:
			a.Value = slog.StringValue("TRACE")
		case levelFatal:
			a.Value = slog.StringValue("FATAL")
		}
	}
	return a
}
