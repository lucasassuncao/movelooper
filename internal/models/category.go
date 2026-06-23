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
// A category must have enabled: true set explicitly; omitting the field disables it.
func (c *Category) IsEnabled() bool {
	return c.Enabled != nil && *c.Enabled
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

// CategoryFilter holds the optional filtering rules applied to files before they are moved.
// At the top level it behaves as an implicit AND: all populated sub-fields must pass.
// Use any/all/not for explicit boolean composition.
type CategoryFilter struct {
	Match *MatchFilter     `yaml:"match" mapstructure:"match"`
	Age   *AgeFilter       `yaml:"age"   mapstructure:"age"`
	Size  *SizeFilter      `yaml:"size"  mapstructure:"size"`
	Any   []CategoryFilter `yaml:"any"   mapstructure:"any"`
	All   []CategoryFilter `yaml:"all"   mapstructure:"all"`
	Not   []CategoryFilter `yaml:"not"   mapstructure:"not"`
}

// MatchFilter constrains by filename: one of literal, regex, or glob (mutually exclusive).
type MatchFilter struct {
	Literal       string         `yaml:"literal"        mapstructure:"literal"`
	Regex         string         `yaml:"regex"          mapstructure:"regex"`
	Glob          string         `yaml:"glob"           mapstructure:"glob"`
	CaseSensitive bool           `yaml:"case-sensitive" mapstructure:"case-sensitive"`
	CompiledRegex *regexp.Regexp `yaml:"-"              mapstructure:"-"`
}

// AgeFilter constrains by modification time.
type AgeFilter struct {
	Min time.Duration `yaml:"min" mapstructure:"min"`
	Max time.Duration `yaml:"max" mapstructure:"max"`
}

// SizeFilter constrains by file size.
type SizeFilter struct {
	Min      string `yaml:"min" mapstructure:"min"`
	Max      string `yaml:"max" mapstructure:"max"`
	MinBytes int64  `yaml:"-"   mapstructure:"-"`
	MaxBytes int64  `yaml:"-"   mapstructure:"-"`
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
			Description: "Whether this category is active. Must be explicitly set to true; omitting this field disables the category.",
			Default:     "false",
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
			Description: "Absolute paths to skip during recursive walk. The destination path is always auto-excluded.",
			Example:     "exclude-paths:\n  - /home/user/Downloads/archives\n  - /home/user/Downloads/.Trash",
		}},
		"filter": {
			FieldMeta: editor.FieldMeta{
				Description: "Optional filtering rules applied to each matched file. All populated sub-fields must match (AND logic) unless any/all are used.",
			},
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
			Example:     "rename: \"{year}-{month}-{day}_{name}\"\n\n# Available tokens:\n# {name}  original filename without extension\n# {ext}   original extension\n# {year}, {month}, {day}, {hour}, {minute}, {second}\n# {seq}   auto-incrementing counter\n# {sha256:N}  first N hex chars of SHA-256",
		}},
	}
}

func (CategoryFilter) Metadata() map[string]*metadata.Node {
	anyNode := &metadata.Node{FieldMeta: editor.FieldMeta{
		Description: "OR logic: file must match at least one sub-filter.",
		MinCount:    1,
		Example:     "any:\n  - match:\n      glob: \"invoice_*\"\n  - match:\n      glob: \"receipt_*\"",
	}}
	allNode := &metadata.Node{FieldMeta: editor.FieldMeta{
		Description: "AND logic: file must match all sub-filters simultaneously.",
		MinCount:    1,
		Example:     "all:\n  - size:\n      min: 100KB\n  - age:\n      max: 168h",
	}}
	notNode := &metadata.Node{FieldMeta: editor.FieldMeta{
		Description: "NOT logic: exclude files matching any of these sub-filters.",
		Example:     "not:\n  - match:\n      glob: \"*_draft*\"",
	}}
	children := map[string]*metadata.Node{
		"match": {FieldMeta: editor.FieldMeta{
			Description: "Name-based filter: glob, regex, or literal match (pick one).",
		}},
		"age": {FieldMeta: editor.FieldMeta{
			Description: "Modification-time constraints.",
		}},
		"size": {FieldMeta: editor.FieldMeta{
			Description: "File-size constraints.",
		}},
		"any": anyNode,
		"all": allNode,
		"not": notNode,
	}
	anyNode.Children = children
	allNode.Children = children
	notNode.Children = children
	return children
}

func (MatchFilter) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"literal": {FieldMeta: editor.FieldMeta{
			Description: "Exact filename match (whole name must equal this string). Mutually exclusive with regex and glob.",
			Example:     "literal: \"Anna's Archive.pdf\"",
		}},
		"regex": {FieldMeta: editor.FieldMeta{
			Description: "RE2 regular expression matched against the filename. Mutually exclusive with glob and literal.",
			Formats:     []editor.Format{FormatRegex},
			Example:     "regex: \"^\\d{4}-\\d{2}-\\d{2}_.*\\.pdf$\"",
		}},
		"glob": {FieldMeta: editor.FieldMeta{
			Description: "Glob pattern matched against the filename. Mutually exclusive with regex and literal.",
			Formats:     []editor.Format{FormatGlob},
			Example:     "glob: \"screenshot_*\"",
		}},
		"case-sensitive": {FieldMeta: editor.FieldMeta{
			Description: "Whether matching is case-sensitive. Applies to regex, glob, and literal.",
			Default:     "false",
			Example:     "case-sensitive: false",
		}},
	}
}

func (AgeFilter) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"min": {FieldMeta: editor.FieldMeta{
			Description: "Only match files older than this duration.",
			Min:         "0s",
			Max:         "87600h",
			Formats:     []editor.Format{editor.FormatDuration},
			Example:     "min: 24h",
		}},
		"max": {FieldMeta: editor.FieldMeta{
			Description: "Only match files newer than this duration.",
			Min:         "0s",
			Max:         "87600h",
			Formats:     []editor.Format{editor.FormatDuration},
			Example:     "max: 720h",
		}},
	}
}

func (SizeFilter) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"min": {FieldMeta: editor.FieldMeta{
			Description: "Only match files at least this large. KB/MB/GB/TB are decimal; KiB/MiB/GiB/TiB are binary.",
			Min:         "0B",
			Max:         "100TB",
			Example:     "min: 1MB",
		}},
		"max": {FieldMeta: editor.FieldMeta{
			Description: "Only match files no larger than this size.",
			Min:         "0B",
			Max:         "100TB",
			Example:     "max: 50GB",
		}},
	}
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
			Description: "Shell interpreter for hook commands. Defaults to $SHELL on Unix/macOS and cmd on Windows.",
			Example:     "shell: bash",
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
			Example:     "run:\n  - echo \"before: $ML_SOURCE_PATH\"",
		}},
	}
}
