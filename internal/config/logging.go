package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/logger"
)

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

// ConfigureLogger builds the logger the configuration asks for: bezel's tree
// in th, or JSON lines when the format is "json". formatOverride (--format)
// wins over configuration.logging.format unless empty; showCaller (the hidden
// --show-caller) adds each line's source. The Closer is non-nil only when
// writing to a file, and must be called on exit.
func ConfigureLogger(k *koanf.Koanf, formatOverride string, showCaller bool, th theme.Resolved) (*slog.Logger, io.Closer, error) {
	w, closer, err := logWriterFactory(k.String("configuration.logging.output")).Writer(k)
	if err != nil {
		return nil, nil, err
	}

	level := k.String("configuration.logging.level")
	format := k.String("configuration.logging.format")
	if formatOverride != "" {
		format = formatOverride
	}
	if format == "json" {
		return logger.NewJSON(w, level, showCaller), closer, nil
	}
	color := k.String("configuration.logging.color")
	return logger.NewPretty(w, th, level, showCaller, color, k.Int("configuration.logging.max-width")), closer, nil
}

// LogTheme is the bezel theme called name; empty is bezel's default.
func LogTheme(name string) (theme.Theme, error) {
	t, err := theme.Lookup(name)
	if err != nil {
		return theme.Theme{}, fmt.Errorf("log theme: %w: run 'movelooper --list-themes' to see available themes", err)
	}
	return t, nil
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

	rotateLog(file, maxLogBytes)
	logFile, err := os.OpenFile(filepath.Clean(file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the log file: %w", err)
	}

	return logFile, nil
}

// maxLogBytes is the size past which the log starts over.
const maxLogBytes = 10 << 20

// rotateLog moves a log past limit bytes to <path>.1, replacing the older
// copy, so the log keeps two files at most. A failure leaves it growing.
func rotateLog(path string, limit int64) {
	if info, err := os.Stat(path); err == nil && info.Size() > limit {
		_ = os.Rename(path, path+".1")
	}
}
