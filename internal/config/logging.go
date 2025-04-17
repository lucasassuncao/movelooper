package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

const maxWidth = 70

// ConfigureLogger configures the logger based on the configuration
func ConfigureLogger(v *viper.Viper) (*pterm.Logger, error) {
	switch v.GetString("configuration.output") {
	default:
		fallthrough
	case "console":
		return configurePTermLogger(v)
	case "file", "log":
		return configureFileLogger(v)
	case "both":
		return configureMultiWriterLogger(v)
	}
}

// configurePTermLogger configures the logger to write to the console
func configurePTermLogger(v *viper.Viper) (*pterm.Logger, error) {
	l := v.GetString("configuration.log-level")
	s := v.GetBool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(os.Stdout).WithMaxWidth(maxWidth), nil
}

// configureFileLogger configures the logger to write to a file
func configureFileLogger(v *viper.Viper) (*pterm.Logger, error) {
	f, err := openLogFile(v)
	if err != nil {
		return nil, err
	}

	l := v.GetString("configuration.log-level")
	s := v.GetBool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(f).WithMaxWidth(maxWidth), nil
}

// configureMultiWriterLogger configures the logger to write to both the console and a file
func configureMultiWriterLogger(v *viper.Viper) (*pterm.Logger, error) {
	f, err := openLogFile(v)
	if err != nil {
		return nil, err
	}

	l := v.GetString("configuration.log-level")
	s := v.GetBool("configuration.show-caller")

	multiWriter := io.MultiWriter(os.Stdout, f)

	return pterm.DefaultLogger.WithCaller(s).WithLevel(parseLogLevel(l)).WithWriter(multiWriter).WithMaxWidth(maxWidth), nil
}

// openLogFile opens the log file for writing
func openLogFile(v *viper.Viper) (*os.File, error) {
	file := v.GetString("configuration.log-file")

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
