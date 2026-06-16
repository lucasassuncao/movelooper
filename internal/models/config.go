package models

import (
	"time"

	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/metadata"
)

// Config represents the complete structure of the movelooper.yaml file
type Config struct {
	Configuration Configuration `yaml:"configuration" mapstructure:"configuration"`
	Categories    []Category    `yaml:"categories" mapstructure:"categories"`
}

// Configuration holds the general settings for Movelooper
type Configuration struct {
	Output       string        `yaml:"output" mapstructure:"output"`
	LogFile      string        `yaml:"log-file" mapstructure:"log-file"`
	LogLevel     string        `yaml:"log-level" mapstructure:"log-level"`
	ShowCaller   bool          `yaml:"show-caller" mapstructure:"show-caller"`
	WatchDelay   time.Duration `yaml:"watch-delay" mapstructure:"watch-delay"`
	HistoryLimit int           `yaml:"history-limit" mapstructure:"history-limit"`
	HistoryFile  string        `yaml:"history-file" mapstructure:"history-file"`
}

func (Config) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"configuration": {FieldMeta: editor.FieldMeta{
			Description: "General settings for movelooper: logging output, watch interval, and history size.",
			Required:    true,
		}},
		"categories": {FieldMeta: editor.FieldMeta{
			Description: "List of file movement rules. Each entry defines a source directory, file filters, a destination, and optional hooks.",
			Required:    true,
		}},
	}
}

func (Configuration) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"output": {FieldMeta: editor.FieldMeta{
			Description: "Where log output is written. Use 'both' to write to the console and a file simultaneously. 'log' is an alias for 'file'.",
			Required:    true,
			OneOf:       []string{"console", "file", "log", "both"},
			Default:     "console",
			Example:     "output: console",
		}},
		"log-file": {FieldMeta: editor.FieldMeta{
			Description: "Path to the log file. Only used when output is 'file' or 'both'. Supports ~ for the home directory.",
			Default:     "~/movelooper.log",
			Formats:     []editor.Format{editor.FormatDirectoryPath},
			Example:     "log-file: ~/movelooper.log",
		}},
		"log-level": {FieldMeta: editor.FieldMeta{
			Description: "Minimum severity level to emit. Lower levels produce more output; 'fatal' produces the least.",
			Required:    true,
			OneOf:       []string{"trace", "debug", "info", "warn", "error", "fatal"},
			Default:     "info",
			Example:     "log-level: info",
		}},
		"show-caller": {FieldMeta: editor.FieldMeta{
			Description: "Append the source file and line number to each log entry. Useful when debugging hooks or scanners.",
			Default:     "false",
			Example:     "show-caller: false",
		}},
		"watch-delay": {FieldMeta: editor.FieldMeta{
			Description: "Interval between directory scans in watch mode. Accepts Go duration strings (e.g. 30s, 5m, 1h).",
			Default:     "5m",
			Min:         "1s",
			Max:         "168h",
			Formats:     []editor.Format{editor.FormatDuration},
			Example:     "watch-delay: 5m",
		}},
		"history-limit": {FieldMeta: editor.FieldMeta{
			Description: "Maximum number of move events kept in the undo history. Older entries are evicted when the limit is reached.",
			Default:     "100",
			Min:         "1",
			Max:         "100000",
			Example:     "history-limit: 100",
		}},
		"history-file": {FieldMeta: editor.FieldMeta{
			Description: "Path to the history file used for undo. Defaults to ~/.movelooper/history/movelooper.json when not set.",
			Default:     "~/.movelooper/history/movelooper.json",
			Formats:     []editor.Format{editor.FormatDirectoryPath},
			Example:     "history-file: ~/.movelooper/history/movelooper.json",
		}},
	}
}
