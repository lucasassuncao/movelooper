package models

import (
	"regexp"
	"time"
)

// ConflictStrategy defines what happens when a destination file already exists.
type ConflictStrategy string

const (
	ConflictStrategyRename    ConflictStrategy = "rename"
	ConflictStrategyHashCheck ConflictStrategy = "hash_check"
	ConflictStrategyOverwrite ConflictStrategy = "overwrite"
	ConflictStrategySkip      ConflictStrategy = "skip"
	ConflictStrategyNewest    ConflictStrategy = "newest"
	ConflictStrategyOldest    ConflictStrategy = "oldest"
	ConflictStrategyLarger    ConflictStrategy = "larger"
	ConflictStrategySmaller   ConflictStrategy = "smaller"
)

// Action defines the file operation to perform when moving a category.
type Action string

const (
	ActionMove    Action = "move"
	ActionCopy    Action = "copy"
	ActionSymlink Action = "symlink"
)

// Category represents a file category with its properties
type Category struct {
	Name        string              `yaml:"name" mapstructure:"name"`
	Enabled     *bool               `yaml:"enabled" mapstructure:"enabled"`
	Source      CategorySource      `yaml:"source" mapstructure:"source"`
	Destination CategoryDestination `yaml:"destination" mapstructure:"destination"`
	Hooks       *CategoryHooks      `yaml:"hooks" mapstructure:"hooks"`
}

// IsEnabled reports whether the category is active.
// A category is enabled when the field is omitted (nil) or explicitly set to true.
func (c *Category) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// CategorySource holds the source path, extensions, and filters for a category
type CategorySource struct {
	Path         string         `yaml:"path"          mapstructure:"path"`
	Extensions   []string       `yaml:"extensions"    mapstructure:"extensions"`
	Filter       CategoryFilter `yaml:"filter"        mapstructure:"filter"`
	Recursive    bool           `yaml:"recursive"     mapstructure:"recursive"`
	MaxDepth     int            `yaml:"max-depth"     mapstructure:"max-depth"`
	ExcludePaths []string       `yaml:"exclude-paths" mapstructure:"exclude-paths"`
}

// CategoryDestination holds the destination path and placement rules for a category
type CategoryDestination struct {
	Path             string           `yaml:"path" mapstructure:"path"`
	OrganizeBy       string           `yaml:"organize-by" mapstructure:"organize-by"`
	ConflictStrategy ConflictStrategy `yaml:"conflict-strategy" mapstructure:"conflict-strategy"`
	Action           Action           `yaml:"action" mapstructure:"action"`
	Rename           string           `yaml:"rename" mapstructure:"rename"`
}

// CategoryFilter holds the optional filtering rules for a category
type CategoryFilter struct {
	Regex         string           `yaml:"regex" mapstructure:"regex"`
	Glob          string           `yaml:"glob" mapstructure:"glob"`
	Include       []string         `yaml:"include" mapstructure:"include"`
	Ignore        []string         `yaml:"ignore" mapstructure:"ignore"`
	CaseSensitive bool             `yaml:"case-sensitive" mapstructure:"case-sensitive"`
	MinAge        time.Duration    `yaml:"min-age" mapstructure:"min-age"`
	MaxAge        time.Duration    `yaml:"max-age" mapstructure:"max-age"`
	MinSize       string           `yaml:"min-size" mapstructure:"min-size"`
	MaxSize       string           `yaml:"max-size" mapstructure:"max-size"`
	CompiledRegex *regexp.Regexp   `yaml:"-" mapstructure:"-"` // compiled from Regex
	MinSizeBytes  int64            `yaml:"-" mapstructure:"-"` // parsed from MinSize
	MaxSizeBytes  int64            `yaml:"-" mapstructure:"-"` // parsed from MaxSize
	Any           []CategoryFilter `yaml:"any" mapstructure:"any"`
	All           []CategoryFilter `yaml:"all" mapstructure:"all"`
}

// CategoryHooks holds optional before/after hooks for a category.
type CategoryHooks struct {
	Before *CategoryHook `yaml:"before" mapstructure:"before"`
	After  *CategoryHook `yaml:"after" mapstructure:"after"`
}

// CategoryHook defines a list of shell commands to run at a lifecycle point.
type CategoryHook struct {
	Shell     string   `yaml:"shell" mapstructure:"shell"`
	OnFailure string   `yaml:"on-failure" mapstructure:"on-failure"`
	Run       []string `yaml:"run" mapstructure:"run"`
}
