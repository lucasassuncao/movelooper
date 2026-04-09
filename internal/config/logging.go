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

// ConfigureLogger configures the logger based on the configuration.
// Returns the logger, a Closer that must be called on exit (non-nil only when
// writing to a file), and any error.
func ConfigureLogger(k *koanf.Koanf) (*pterm.Logger, io.Closer, error) {
	switch k.String("configuration.output") {
	default:
		fallthrough
	case "console":
		return configurePTermLogger(k)
	case "file", "log":
		return configureFileLogger(k)
	case "both":
		return configureMultiWriterLogger(k)
	}
}

// configurePTermLogger configures the logger to write to the console
func configurePTermLogger(k *koanf.Koanf) (*pterm.Logger, io.Closer, error) {
	l := k.String("configuration.log-level")
	s := k.Bool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(os.Stdout).WithMaxWidth(maxWidth), nil, nil
}

// configureFileLogger configures the logger to write to a file
func configureFileLogger(k *koanf.Koanf) (*pterm.Logger, io.Closer, error) {
	f, err := openLogFile(k)
	if err != nil {
		return nil, nil, err
	}

	l := k.String("configuration.log-level")
	s := k.Bool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(f).WithMaxWidth(maxWidth), f, nil
}

// configureMultiWriterLogger configures the logger to write to both the console and a file
func configureMultiWriterLogger(k *koanf.Koanf) (*pterm.Logger, io.Closer, error) {
	f, err := openLogFile(k)
	if err != nil {
		return nil, nil, err
	}

	l := k.String("configuration.log-level")
	s := k.Bool("configuration.show-caller")

	multiWriter := io.MultiWriter(os.Stdout, f)

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(multiWriter).WithMaxWidth(maxWidth), f, nil
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
