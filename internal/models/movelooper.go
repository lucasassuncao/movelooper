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

// IsEnabled reports whether the category is active.
// A category is enabled when the field is omitted (nil) or explicitly set to true.
func (c *Category) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
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
	Include       []string       `yaml:"include" mapstructure:"include"`
	Ignore        []string       `yaml:"ignore" mapstructure:"ignore"`
	CaseSensitive bool           `yaml:"case-sensitive" mapstructure:"case-sensitive"`
	MinAge        time.Duration  `yaml:"min-age" mapstructure:"min-age"`
	MaxAge        time.Duration  `yaml:"max-age" mapstructure:"max-age"`
	MinSize       string         `yaml:"min-size" mapstructure:"min-size"`
	MaxSize       string         `yaml:"max-size" mapstructure:"max-size"`
	CompiledRegex *regexp.Regexp `yaml:"-" mapstructure:"-"` // compiled from Regex
	MinSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MinSize
	MaxSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MaxSize
}

// CategorySource holds the source path, extensions, and filters for a category
type CategorySource struct {
	Path       string         `yaml:"path" mapstructure:"path"`
	Extensions []string       `yaml:"extensions" mapstructure:"extensions"`
	Filter     CategoryFilter `yaml:"filter" mapstructure:"filter"`
}

// CategoryDestination holds the destination path and placement rules for a category
type CategoryDestination struct {
	Path             string `yaml:"path" mapstructure:"path"`
	OrganizeBy       string `yaml:"organize-by" mapstructure:"organize-by"`
	ConflictStrategy string `yaml:"conflict-strategy" mapstructure:"conflict-strategy"`
}

// Category represents a file category with its properties
type Category struct {
	Name        string              `yaml:"name" mapstructure:"name"`
	Enabled     *bool               `yaml:"enabled" mapstructure:"enabled"`
	Source      CategorySource      `yaml:"source" mapstructure:"source"`
	Destination CategoryDestination `yaml:"destination" mapstructure:"destination"`
}
