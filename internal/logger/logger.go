package logger

import "github.com/pterm/pterm"

// Logger is the minimal logging interface used across the application.
// *pterm.Logger satisfies this interface.
type Logger interface {
	Trace(msg string, args ...[]pterm.LoggerArgument)
	Debug(msg string, args ...[]pterm.LoggerArgument)
	Info(msg string, args ...[]pterm.LoggerArgument)
	Warn(msg string, args ...[]pterm.LoggerArgument)
	Error(msg string, args ...[]pterm.LoggerArgument)
	Args(args ...interface{}) []pterm.LoggerArgument
}
