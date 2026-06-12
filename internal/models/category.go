package models

import (
	"regexp"
	"time"

	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/metadata"
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

func (Category) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"name": {FieldMeta: editor.FieldMeta{
			Description: "Human-readable identifier for this category. Used in logs, history, and the --category filter flag.",
			Required:    true,
			Example:     "name: screenshots",
		}},
		"enabled": {FieldMeta: editor.FieldMeta{
			Description: "Whether this category is active. Set to false to pause without deleting the entry.",
			Default:     "true",
			Example:     "enabled: true",
		}},
		"source": {FieldMeta: editor.FieldMeta{
			Description: "Source directory configuration: which path to watch, which extensions to include, and how deep to scan.",
			Required:    true,
		}},
		"destination": {FieldMeta: editor.FieldMeta{
			Description: "Destination configuration: where to place matched files, how to name them, and what to do on conflicts.",
			Required:    true,
		}},
		"hooks": {FieldMeta: editor.FieldMeta{
			Description: "Optional shell commands to run before and after each file is moved.",
		}},
	}
}

func (CategorySource) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"path": {FieldMeta: editor.FieldMeta{
			Description: "Directory to watch for incoming files.",
			Required:    true,
			Formats:     []editor.Format{editor.FormatDirectoryPath},
			Example:     "path: ~/Downloads",
		}},
		"extensions": {FieldMeta: editor.FieldMeta{
			Description: "File extensions to match (without the leading dot). Use the special value \"all\" to match every file.",
			Required:    true,
			MinCount:    1,
			Unique:      true,
			Example:     "extensions:\n  - jpg\n  - jpeg\n  - png\n\n# or, to match every file:\nextensions: [all]",
		}},
		"recursive": {FieldMeta: editor.FieldMeta{
			Description: "Whether to scan sub-directories of the source path. Combine with max-depth to limit depth.",
			Default:     "false",
			Example:     "recursive: true",
		}},
		"max-depth": {FieldMeta: editor.FieldMeta{
			Description: "Maximum sub-directory depth when recursive is true. 0 means unlimited.",
			Default:     "0",
			Min:         "0",
			Max:         "256",
			Example:     "max-depth: 3",
		}},
		"exclude-paths": {FieldMeta: editor.FieldMeta{
			Description: "Sub-paths (relative to the source path) to skip during scanning.",
			Example:     "exclude-paths:\n  - tmp\n  - .Trash",
		}},
		"filter": {
			FieldMeta: editor.FieldMeta{
				Description: "Optional filtering rules applied to each matched file. All populated sub-fields must match (AND logic) unless any/all are used.",
			},
			Children: categoryFilterChildren(),
		},
	}
}

func (CategoryDestination) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"path": {FieldMeta: editor.FieldMeta{
			Description: "Directory where matched files are placed.",
			Required:    true,
			Formats:     []editor.Format{editor.FormatDirectoryPath},
			Example:     "path: ~/Pictures/Sorted",
		}},
		"organize-by": {FieldMeta: editor.FieldMeta{
			Description: "Token pattern used to build sub-directories inside the destination path. Leave empty to place all files directly.",
			Formats:     []editor.Format{FormatOrganizeByPattern},
			Example:     "organize-by: \"{ext}/{year}\"\n\n# Available tokens:\n# {ext}   file extension\n# {year}  4-digit year\n# {month} 2-digit month\n# {day}   2-digit day",
		}},
		"conflict-strategy": {FieldMeta: editor.FieldMeta{
			Description: "What to do when a file with the same name already exists at the destination.",
			OneOf:       []string{"rename", "hash_check", "overwrite", "skip", "newest", "oldest", "larger", "smaller"},
			Default:     "rename",
			Example:     "conflict-strategy: rename",
		}},
		"action": {FieldMeta: editor.FieldMeta{
			Description: "File operation to perform. 'move' removes the source; 'copy' keeps it; 'symlink' creates a symbolic link.",
			OneOf:       []string{"move", "copy", "symlink"},
			Default:     "move",
			Example:     "action: move",
		}},
		"rename": {FieldMeta: editor.FieldMeta{
			Description: "Token pattern for the destination filename (without extension). Leave empty to keep the original name.",
			Formats:     []editor.Format{FormatRenamePattern},
			Example:     "rename: \"{year}-{month}-{day}_{name}\"\n\n# Available tokens:\n# {name}  original filename without extension\n# {ext}   original extension\n# {year}, {month}, {day}, {hour}, {min}, {sec}\n# {seq}   auto-incrementing counter\n# {hash}  SHA-256 prefix of the file content",
		}},
	}
}

// categoryFilterChildren builds the shared-pointer child map for CategoryFilter.
// CategoryFilter.Any and CategoryFilter.All are both []CategoryFilter (recursive),
// so their nodes point back to the same map — mirrors how the old hints.filterHintChildren worked.
func categoryFilterChildren() map[string]*metadata.Node {
	anyNode := &metadata.Node{
		FieldMeta: editor.FieldMeta{
			Description: "OR logic: file must match at least one sub-filter.",
			Example:     "any:\n  - regex: \"^invoice_.*\"\n  - glob: \"receipt_*\"",
		},
	}
	allNode := &metadata.Node{
		FieldMeta: editor.FieldMeta{
			Description: "AND logic: file must match all sub-filters simultaneously.",
			Example:     "all:\n  - min-size: 100KB\n  - max-age: 168h",
		},
	}
	children := map[string]*metadata.Node{
		"regex": {FieldMeta: editor.FieldMeta{
			Description: "RE2 regular expression matched against the filename (without path). Mutually exclusive with glob.",
			Formats:     []editor.Format{FormatRegex},
			Example:     "regex: \"^\\d{4}-\\d{2}-\\d{2}_.*\\.pdf$\"",
		}},
		"glob": {FieldMeta: editor.FieldMeta{
			Description: "Glob pattern matched against the filename (without path). Mutually exclusive with regex.",
			Formats:     []editor.Format{FormatGlob},
			Example:     "glob: \"screenshot_*\"",
		}},
		"include": {FieldMeta: editor.FieldMeta{
			Description: "Filenames must match at least one of these glob patterns.",
			Example:     "include:\n  - \"report_*\"\n  - \"invoice_*\"",
		}},
		"ignore": {FieldMeta: editor.FieldMeta{
			Description: "Filenames matching these patterns are excluded. Takes precedence over include.",
			Example:     "ignore:\n  - \"*_draft*\"\n  - \"*_temp*\"",
		}},
		"case-sensitive": {FieldMeta: editor.FieldMeta{
			Description: "Whether extension and glob/include/ignore matching is case-sensitive.",
			Default:     "false",
			Example:     "case-sensitive: false",
		}},
		"min-age": {FieldMeta: editor.FieldMeta{
			Description: "Only match files older than this duration. Accepts Go duration strings (e.g. 24h, 168h).",
			Min:         "0s",
			Max:         "87600h",
			Formats:     []editor.Format{editor.FormatDuration},
			Example:     "min-age: 24h",
		}},
		"max-age": {FieldMeta: editor.FieldMeta{
			Description: "Only match files newer than this duration.",
			Min:         "0s",
			Max:         "87600h",
			Formats:     []editor.Format{editor.FormatDuration},
			Example:     "max-age: 720h",
		}},
		"min-size": {FieldMeta: editor.FieldMeta{
			Description: "Only match files at least this large. Accepts human-readable sizes — KB/MB/GB/TB are decimal (powers of 1000), KiB/MiB/GiB/TiB are binary (powers of 1024).",
			Min:         "0B",
			Max:         "100TB",
			Example:     "min-size: 1MB",
		}},
		"max-size": {FieldMeta: editor.FieldMeta{
			Description: "Only match files no larger than this size. Same units as min-size.",
			Min:         "0B",
			Max:         "100TB",
			Example:     "max-size: 50GB",
		}},
		"any": anyNode,
		"all": allNode,
	}
	anyNode.Children = children
	allNode.Children = children
	return children
}

func (CategoryHooks) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"before": {FieldMeta: editor.FieldMeta{
			Description: "Hook executed before the file operation. If it fails, the move is aborted (unless on-failure is 'warn').",
		}},
		"after": {FieldMeta: editor.FieldMeta{
			Description: "Hook executed after the file operation completes successfully.",
		}},
	}
}

func (CategoryHook) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"shell": {FieldMeta: editor.FieldMeta{
			Description: "Shell interpreter for hook commands.",
			Default:     "/bin/sh",
			Example:     "shell: /bin/bash",
		}},
		"on-failure": {FieldMeta: editor.FieldMeta{
			Description: "What to do if a hook command exits non-zero: abort the file's operation, or warn and continue.",
			Required:    true,
			OneOf:       []string{"abort", "warn"},
			Default:     "abort",
			Example:     "on-failure: abort",
		}},
		"run": {FieldMeta: editor.FieldMeta{
			Description: "Shell commands executed in order.",
			Required:    true,
			MinCount:    1,
			Example:     "run:\n  - echo \"before: {{.Source}}\"",
		}},
	}
}
