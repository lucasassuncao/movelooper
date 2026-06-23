package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/pterm/pterm"
)

const maxWidth = 70

// writerBuilder is an interface for building log writers based on the configuration.
type writerBuilder interface {
	Writer(k *koanf.Koanf) (io.Writer, io.Closer, error)
}

type consoleStrategy struct{}
type fileStrategy struct{}
type multiStrategy struct{}

func (consoleStrategy) Writer(_ *koanf.Koanf) (io.Writer, io.Closer, error) {
	return os.Stdout, nil, nil
}

func (fileStrategy) Writer(k *koanf.Koanf) (io.Writer, io.Closer, error) {
	f, err := openLogFile(k)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func (multiStrategy) Writer(k *koanf.Koanf) (io.Writer, io.Closer, error) {
	f, err := openLogFile(k)
	if err != nil {
		return nil, nil, err
	}
	return io.MultiWriter(os.Stdout, f), f, nil
}

var logWriterStrategies = map[string]writerBuilder{
	"console": consoleStrategy{},
	"file":    fileStrategy{},
	"log":     fileStrategy{},
	"both":    multiStrategy{},
}

// logWriterFactory returns the strategy for the given output mode.
func logWriterFactory(output string) writerBuilder {
	if s, ok := logWriterStrategies[output]; ok {
		return s
	}
	return consoleStrategy{}
}

// ConfigureLogger configures the logger based on the configuration.
// It returns a pretty (pterm) logger by default, or a structured JSON (slog)
// logger when logging.format is "json". The Closer must be called on exit
// (non-nil only when writing to a file).
func ConfigureLogger(k *koanf.Koanf) (logger.Logger, io.Closer, error) {
	output := k.String("configuration.logging.output")
	strategy := logWriterFactory(output)

	w, closer, err := strategy.Writer(k)
	if err != nil {
		return nil, nil, err
	}

	level := k.String("configuration.logging.level")
	showCaller := k.Bool("configuration.logging.show-caller")

	if k.String("configuration.logging.format") == "json" {
		// Disable pterm color so any color helpers used in message strings stay
		// inert, keeping the structured JSON output free of ANSI escape codes.
		applyColor("never", output)
		return logger.NewSlog(w, level, showCaller), closer, nil
	}

	applyColor(k.String("configuration.logging.color"), output)

	width := k.Int("configuration.logging.max-width")
	if width <= 0 {
		width = maxWidth
	}

	plog := pterm.DefaultLogger.
		WithCaller(showCaller).
		WithLevel(parseLogLevel(level)).
		WithWriter(w).
		WithMaxWidth(width)

	return plog, closer, nil
}

// colorMu serializes the pterm color toggle, which mutates process-global
// state. ConfigureLogger runs once in production, but tests configure loggers
// concurrently; the lock keeps that access race-free.
var colorMu sync.Mutex

// applyColor toggles pterm's global ANSI styling for the pretty format.
// "auto" keeps color only for pure console output, so file logs stay clean.
func applyColor(mode, output string) {
	colorMu.Lock()
	defer colorMu.Unlock()
	switch mode {
	case "never":
		pterm.DisableColor()
	case "always":
		pterm.EnableColor()
	default: // auto, and any unrecognized value
		if output == "console" {
			pterm.EnableColor()
		} else {
			pterm.DisableColor()
		}
	}
}

func defaultLogFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "movelooper", "logs", "movelooper.log")
	}
	return filepath.Join(homeDir, ".movelooper", "logs", "movelooper.log")
}

// openLogFile opens the log file for writing
func openLogFile(k *koanf.Koanf) (*os.File, error) {
	file := ExpandTilde(k.String("configuration.logging.file"))
	if file == "" {
		file = defaultLogFilePath()
	}

	dir := filepath.Dir(file)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("couldn't create log directory: %w", err)
	}

	logFile, err := os.OpenFile(filepath.Clean(file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the log file: %w", err)
	}

	return logFile, nil
}

// parseLogLevel returns the pterm log level based on the string level
func parseLogLevel(level string) pterm.LogLevel {
	switch level {
	case "trace":
		return pterm.LogLevelTrace
	case "debug":
		return pterm.LogLevelDebug
	case "info":
		return pterm.LogLevelInfo
	case "warn", "warning":
		return pterm.LogLevelWarn
	case "error":
		return pterm.LogLevelError
	case "fatal":
		return pterm.LogLevelFatal
	default:
		return pterm.LogLevelInfo
	}
}
