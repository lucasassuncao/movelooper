package models

import (
	"io"
	"regexp"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/pterm/pterm"
)

// Movelooper holds the app dependencies and runtime state.
// Viper is intentionally absent: it is used only during initialisation
// in preRunHandler and discarded afterwards.
type Movelooper struct {
	Logger     *pterm.Logger
	Config     Configuration
	Categories []*Category
	History    *history.History
	LogCloser  io.Closer // non-nil when logging to a file; closed on exit
}

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
}

// CategoryFilter holds the optional filtering rules for a category
type CategoryFilter struct {
	Regex         string         `yaml:"regex" mapstructure:"regex"`
	Glob          string         `yaml:"glob" mapstructure:"glob"`
	Ignore        []string       `yaml:"ignore" mapstructure:"ignore"`
	MinAge        time.Duration  `yaml:"min-age" mapstructure:"min-age"`
	MaxAge        time.Duration  `yaml:"max-age" mapstructure:"max-age"`
	MinSize       string         `yaml:"min-size" mapstructure:"min-size"`
	MaxSize       string         `yaml:"max-size" mapstructure:"max-size"`
	CompiledRegex *regexp.Regexp `yaml:"-" mapstructure:"-"` // compiled from Regex
	MinSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MinSize
	MaxSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MaxSize
}

// Category represents a file category with its properties
type Category struct {
	Name             string         `yaml:"name" mapstructure:"name"`
	Extensions       []string       `yaml:"extensions" mapstructure:"extensions"`
	Source           string         `yaml:"source" mapstructure:"source"`
	Destination      string         `yaml:"destination" mapstructure:"destination"`
	ConflictStrategy string         `yaml:"conflict-strategy" mapstructure:"conflict-strategy"`
	Filter           CategoryFilter `yaml:"filter" mapstructure:"filter"`
}
