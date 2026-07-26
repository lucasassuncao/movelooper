package models

import (
	"time"

	"github.com/lucasassuncao/yedit/metadata"
	"github.com/lucasassuncao/yedit/spec"
)

// Config represents the complete structure of the movelooper.yaml file
type Config struct {
	Configuration Configuration `yaml:"configuration" mapstructure:"configuration"`
	Categories    []Category    `yaml:"categories" mapstructure:"categories"`
}

// Configuration holds the general settings for Movelooper, grouped into
// logging, watch, history, and defaults sub-sections.
type Configuration struct {
	Logging  Logging   `yaml:"logging" mapstructure:"logging"`
	Watch    Watch     `yaml:"watch" mapstructure:"watch"`
	History  History   `yaml:"history" mapstructure:"history"`
	Defaults *Defaults `yaml:"defaults,omitempty" mapstructure:"defaults"`
}

// Logging holds the log output settings.
type Logging struct {
	Output     string `yaml:"output" mapstructure:"output"`
	Level      string `yaml:"level" mapstructure:"level"`
	File       string `yaml:"file" mapstructure:"file"`
	ShowCaller bool   `yaml:"show-caller" mapstructure:"show-caller"`
	Format     string `yaml:"format,omitempty" mapstructure:"format"`
	Color      string `yaml:"color,omitempty" mapstructure:"color"`
	MaxWidth   int    `yaml:"max-width,omitempty" mapstructure:"max-width"`
}

// Watch holds the watch-mode settings.
type Watch struct {
	Delay        time.Duration `yaml:"delay" mapstructure:"delay"`
	PollInterval time.Duration `yaml:"poll-interval,omitempty" mapstructure:"poll-interval"`
}

// History holds the undo-history settings.
type History struct {
	Limit   int    `yaml:"limit" mapstructure:"limit"`
	File    string `yaml:"file" mapstructure:"file"`
	Enabled bool   `yaml:"enabled,omitempty" mapstructure:"enabled"`
}

// Defaults holds fallback values applied to any category that omits them.
type Defaults struct {
	ConflictStrategy ConflictStrategy `yaml:"conflict-strategy,omitempty" mapstructure:"conflict-strategy"`
	Action           Action           `yaml:"action,omitempty" mapstructure:"action"`
	OrganizeBy       string           `yaml:"organize-by,omitempty" mapstructure:"organize-by"`
}

func (Config) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"configuration": {FieldMeta: spec.FieldMeta{
			Description: "General settings for movelooper, grouped into logging, watch, history, and defaults sub-sections.",
			Required:    true,
		}},
		"categories": {FieldMeta: spec.FieldMeta{
			Description: "List of file movement rules. Each entry defines a source directory, file filters, a destination, and optional hooks.",
			Required:    true,
		}},
	}
}

func (Configuration) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"logging": {FieldMeta: spec.FieldMeta{
			Description: "Log output settings: destination, severity level, format, file path, and caller info.",
			Required:    true,
		}},
		"watch": {FieldMeta: spec.FieldMeta{
			Description: "Watch-mode settings.",
		}},
		"history": {FieldMeta: spec.FieldMeta{
			Description: "Undo-history settings: whether tracking is on, how many batches to keep, and where to store them.",
		}},
		"defaults": {FieldMeta: spec.FieldMeta{
			Description: "Fallback destination settings applied to any category that omits them. Per-category values always win.",
		}},
	}
}

func (Logging) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"output": {FieldMeta: spec.FieldMeta{
			Description: "Where log output is written. Use 'both' to write to the console and a file simultaneously. 'log' is an alias for 'file'.",
			Required:    true,
			OneOf:       []string{"console", "file", "log", "both"},
			Default:     "console",
			Example:     "output: console",
		}},
		"level": {FieldMeta: spec.FieldMeta{
			Description: "Minimum severity level to emit. Lower levels produce more output; 'fatal' produces the least.",
			Required:    true,
			OneOf:       []string{"trace", "debug", "info", "warn", "error", "fatal"},
			Default:     "info",
			Example:     "level: info",
		}},
		"file": {FieldMeta: spec.FieldMeta{
			Description: "Path to the log file. Only used when output is 'file' or 'both'. Supports ~ for the home directory.",
			Default:     "~/movelooper.log",
			Formats:     []spec.Format{spec.FormatDirectoryPath},
			Example:     "file: ~/movelooper.log",
		}},
		"show-caller": {FieldMeta: spec.FieldMeta{
			Description: "Append the source file and line number to each log entry. Useful when debugging hooks or scanners.",
			Default:     "false",
			Example:     "show-caller: false",
		}},
		"format": {FieldMeta: spec.FieldMeta{
			Description: "Log rendering format. 'pretty' is the human-readable console renderer; 'json' emits structured slog JSON lines for log aggregation.",
			OneOf:       []string{"pretty", "json"},
			Default:     "pretty",
			Example:     "format: pretty",
		}},
		"color": {FieldMeta: spec.FieldMeta{
			Description: "ANSI color for the pretty format. 'auto' colors the console but not files; 'always'/'never' force it on or off. Ignored when format is json.",
			OneOf:       []string{"auto", "always", "never"},
			Default:     "auto",
			Example:     "color: auto",
		}},
		"max-width": {FieldMeta: spec.FieldMeta{
			Description: "Maximum width, in columns, for wrapping pretty log lines. Ignored when format is json.",
			Default:     "70",
			Min:         "20",
			Max:         "500",
			Example:     "max-width: 70",
		}},
	}
}

func (Watch) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"delay": {FieldMeta: spec.FieldMeta{
			Description: "How long a file must go without a new create/write event before it is considered stable and moved. Accepts Go duration strings (e.g. 30s, 5m, 1h).",
			Default:     "5m",
			Min:         "1s",
			Max:         "168h",
			Formats:     []spec.Format{spec.FormatDuration},
			Example:     "delay: 5m",
		}},
		"poll-interval": {FieldMeta: spec.FieldMeta{
			Description: "How often watch mode re-checks pending files for stability. Keep it shorter than delay so stable files are picked up promptly.",
			Default:     "5s",
			Min:         "1s",
			Max:         "1h",
			Formats:     []spec.Format{spec.FormatDuration},
			Example:     "poll-interval: 5s",
		}},
	}
}

func (History) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"enabled": {FieldMeta: spec.FieldMeta{
			Description: "Whether move events are recorded for undo. Set to false to skip history tracking entirely.",
			Default:     "true",
			Example:     "enabled: true",
		}},
		"limit": {FieldMeta: spec.FieldMeta{
			Description: "Maximum number of move batches kept in the undo history. Older batches are evicted when the limit is reached.",
			Default:     "100",
			Min:         "1",
			Max:         "100000",
			Example:     "limit: 100",
		}},
		"file": {FieldMeta: spec.FieldMeta{
			Description: "Path to the history file used for undo. Defaults to ~/.movelooper/history/movelooper.json when not set.",
			Default:     "~/.movelooper/history/movelooper.json",
			Formats:     []spec.Format{spec.FormatDirectoryPath},
			Example:     "file: ~/.movelooper/history/movelooper.json",
		}},
	}
}

func (Defaults) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"conflict-strategy": {FieldMeta: spec.FieldMeta{
			Description: "Fallback conflict-strategy for categories that omit destination.conflict-strategy.",
			OneOf:       []string{"rename", "hash_check", "overwrite", "skip", "newest", "oldest", "larger", "smaller"},
			Example:     "conflict-strategy: rename",
		}},
		"action": {FieldMeta: spec.FieldMeta{
			Description: "Fallback action for categories that omit destination.action.",
			OneOf:       []string{"move", "copy", "symlink", "archive"},
			Example:     "action: move",
		}},
		"organize-by": {FieldMeta: spec.FieldMeta{
			Description: "Fallback organize-by template for categories that omit destination.organize-by.",
			Example:     "organize-by: \"{ext}\"",
		}},
	}
}
