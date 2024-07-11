package logging

import (
	"io"
	"log"
	"movelooper/types"
	"os"

	"github.com/logrusorgru/aurora/v4"
)

var Log *Logger

type Logger struct {
	debug   *log.Logger
	info    *log.Logger
	warning *log.Logger
	err     *log.Logger
	writer  io.Writer
}

func GetLogger(prefix string) *Logger {
	// Initialize Logger
	Log, err := NewLogger(prefix, types.LogType)
	if err != nil {
		return nil
	}
	return Log
}

func NewLogger(prefix, typ string) (*Logger, error) {

	var file *os.File
	var writer io.Writer
	var err error

	switch typ {
	case "logs":
		// Open log file for appending, create if it doesn't exist
		file, err = os.OpenFile(types.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		writer = io.Writer(file)
	case "terminal":
		writer = io.Writer(os.Stdout)
	default:
		writer = io.Writer(os.Stdout)
	}

	format := log.Ldate | log.Ltime | log.Lmsgprefix
	logger := log.New(writer, prefix, format)

	return &Logger{
		debug:   log.New(writer, aurora.Sprintf(aurora.Blue("[DEBUG] ")), logger.Flags()),
		info:    log.New(writer, aurora.Sprintf(aurora.Green("[INFO] ")), logger.Flags()),
		warning: log.New(writer, aurora.Sprintf(aurora.Yellow("[WARNING] ")), logger.Flags()),
		err:     log.New(writer, aurora.Sprintf(aurora.Red("[ERROR] ")), logger.Flags()),
		writer:  writer,
	}, nil
}

// Return non-formatted logs
func (l *Logger) Debug(v ...interface{}) {
	l.debug.Println(v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.info.Println(v...)
}

func (l *Logger) Warning(v ...interface{}) {
	l.warning.Println(v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.err.Println(v...)
}

// Return formatted logs
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.warning.Printf(format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.err.Printf(format, v...)
}
