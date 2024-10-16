package logging

import (
	"log"
	"movelooper/types"
	"os"

	"github.com/pterm/pterm"
)

// Exported logger
var Logger *pterm.Logger

// LoggerConfig holds configuration options for the logger
type LoggerConfig struct {
	LogType       string
	LogLevel      pterm.LogLevel
	IncludeCaller bool
}

// ConfigureLogger initializes the logger with the provided configuration
func ConfigureLogger(config LoggerConfig) *pterm.Logger {
	logger := pterm.DefaultLogger.WithLevel(config.LogLevel)

	// Conditionally include caller information
	if config.IncludeCaller || (config.LogLevel == pterm.LogLevelTrace) {
		logger = logger.WithCaller()
	}

	switch config.LogType {
	case "log":
		logFile, err := os.OpenFile(types.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal("Could not open log file. Exiting...")
		}
		logger = logger.WithWriter(logFile)
	case "terminal":
		logger = logger.WithWriter(os.Stdout)
	}

	return logger
}
