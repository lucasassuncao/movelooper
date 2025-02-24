package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

func ConfigureLogger() (*pterm.Logger, error) {
	switch viper.GetString("configuration.output") {
	default:
		fallthrough
	case "console":
		return configurePTermLogger()
	case "file", "log":
		return configureFileLogger()
	}
}

func configurePTermLogger() (*pterm.Logger, error) {
	l := viper.GetString("configuration.log-level")
	s := viper.GetBool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(logLevel(l)).WithWriter(os.Stdout), nil
}

func configureFileLogger() (*pterm.Logger, error) {
	f, err := openLogFile()
	if err != nil {
		return nil, err
	}

	l := viper.GetString("configuration.log-level")
	s := viper.GetBool("configuration.show-caller")

	return pterm.DefaultLogger.WithCaller(s).WithLevel(logLevel(l)).WithWriter(f), nil
}

func openLogFile() (*os.File, error) {
	file := viper.GetString("configuration.log-file")

	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("the log file \"%s\" doesn't exist", file)
	}

	logFile, err := os.OpenFile(filepath.Clean(file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the log file. %w", err)
	}

	return logFile, nil
}

func logLevel(level string) pterm.LogLevel {
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
