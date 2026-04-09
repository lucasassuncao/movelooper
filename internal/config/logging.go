package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/v2"
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
// Returns the logger, a Closer that must be called on exit (non-nil only when
// writing to a file), and any error.
func ConfigureLogger(k *koanf.Koanf) (*pterm.Logger, io.Closer, error) {
	strategy := logWriterFactory(k.String("configuration.output"))

	w, closer, err := strategy.Writer(k)
	if err != nil {
		return nil, nil, err
	}

	logger := pterm.DefaultLogger.
		WithCaller(k.Bool("configuration.show-caller")).
		WithLevel(parseLogLevel(k.String("configuration.log-level"))).
		WithWriter(w).
		WithMaxWidth(maxWidth)

	return logger, closer, nil
}

// openLogFile opens the log file for writing
func openLogFile(k *koanf.Koanf) (*os.File, error) {
	file := k.String("configuration.log-file")
	if file == "" {
		return nil, fmt.Errorf("log-file is required when output is 'file' or 'both'")
	}

	dir := filepath.Dir(file)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("couldn't create log directory: %w", err)
	}

	logFile, err := os.OpenFile(filepath.Clean(file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
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
