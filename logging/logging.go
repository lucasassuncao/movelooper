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
	Log, err := NewLogger(prefix)
	if err != nil {
		return nil
	}
	return Log
}

func NewLogger(prefix string) (*Logger, error) {

	var file *os.File
	var writer io.Writer
	var err error
	var debugFormat, infoFormat, warningFormat, errFormat *log.Logger

	format := log.Ldate | log.Ltime | log.Lmsgprefix
	logger := log.New(writer, prefix, format)

	if types.LogType == "logs" {
		// Open log file for appending, create if it doesn't exist
		file, err = os.OpenFile(types.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		writer = io.Writer(file)

		debugFormat = log.New(writer, "[DEBUG] ", logger.Flags())
		infoFormat = log.New(writer, "[INFO] ", logger.Flags())
		warningFormat = log.New(writer, "[WARNING] ", logger.Flags())
		errFormat = log.New(writer, "[ERROR] ", logger.Flags())
	} else {
		writer = io.Writer(os.Stdout)

		debugFormat = log.New(writer, aurora.Sprintf(aurora.Blue("[DEBUG] ")), logger.Flags())
		infoFormat = log.New(writer, aurora.Sprintf(aurora.Green("[INFO] ")), logger.Flags())
		warningFormat = log.New(writer, aurora.Sprintf(aurora.Yellow("[WARNING] ")), logger.Flags())
		errFormat = log.New(writer, aurora.Sprintf(aurora.Red("[ERROR] ")), logger.Flags())
	}

	return &Logger{
		debug:   debugFormat,
		info:    infoFormat,
		warning: warningFormat,
		err:     errFormat,
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
