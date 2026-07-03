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
	ActionArchive Action = "archive"
)

// Category represents a file category with its properties
type Category struct {
	Name        string              `yaml:"name" mapstructure:"name"`
	Enabled     *bool               `yaml:"enabled" mapstructure:"enabled"`
	Source      CategorySource      `yaml:"source" mapstructure:"source"`
	Destination CategoryDestination `yaml:"destination" mapstructure:"destination"`
	Hooks       *CategoryHooks      `yaml:"hooks,omitempty" mapstructure:"hooks"`
}

// IsEnabled reports whether the category is active.
// A category must have enabled: true set explicitly; omitting the field disables it.
func (c *Category) IsEnabled() bool {
	return c.Enabled != nil && *c.Enabled
}

// CategorySource holds the source path, extensions, and filters for a category
type CategorySource struct {
	Path         string         `yaml:"path"                    mapstructure:"path"`
	Extensions   []string       `yaml:"extensions"              mapstructure:"extensions"`
	Filter       CategoryFilter `yaml:"filter,omitempty"        mapstructure:"filter"`
	Recursive    bool           `yaml:"recursive,omitempty"     mapstructure:"recursive"`
	MaxDepth     int            `yaml:"max-depth,omitempty"     mapstructure:"max-depth"`
	ExcludePaths []string       `yaml:"exclude-paths,omitempty" mapstructure:"exclude-paths"`
}

// CategoryDestination holds the destination path and placement rules for a category
type CategoryDestination struct {
	Path             string           `yaml:"path"                        mapstructure:"path"`
	OrganizeBy       string           `yaml:"organize-by,omitempty"       mapstructure:"organize-by"`
	ConflictStrategy ConflictStrategy `yaml:"conflict-strategy,omitempty" mapstructure:"conflict-strategy"`
	Action           Action           `yaml:"action,omitempty"            mapstructure:"action"`
	Rename           string           `yaml:"rename,omitempty"            mapstructure:"rename"`
	Archive          *ArchiveConfig   `yaml:"archive,omitempty"           mapstructure:"archive"`
}

// ArchiveConfig configures action: archive — how a category's files are packed
// into a single compressed archive at the destination.
type ArchiveConfig struct {
	Format      string `yaml:"format"                mapstructure:"format"`
	Name        string `yaml:"name,omitempty"        mapstructure:"name"`
	Compression string `yaml:"compression,omitempty" mapstructure:"compression"`
	// KeepSource controls whether the original files stay after archiving. Absent
	// means true (keep); set to false to delete the sources after a successful
	// write. Pointer so "unset" is distinguishable from an explicit false.
	KeepSource *bool `yaml:"keep-source,omitempty" mapstructure:"keep-source"`
	Flatten    bool  `yaml:"flatten,omitempty"     mapstructure:"flatten"`
}

// KeepsSource reports whether original files are retained (the default).
func (a *ArchiveConfig) KeepsSource() bool {
	return a.KeepSource == nil || *a.KeepSource
}

// CategoryFilter holds the optional filtering rules applied to files before they are moved.
// At the top level it behaves as an implicit AND: all populated sub-fields must pass.
// Use any/all/not for explicit boolean composition.
type CategoryFilter struct {
	Match *MatchFilter     `yaml:"match,omitempty" mapstructure:"match"`
	Age   *AgeFilter       `yaml:"age,omitempty"   mapstructure:"age"`
	Size  *SizeFilter      `yaml:"size,omitempty"  mapstructure:"size"`
	Mime  string           `yaml:"mime,omitempty"  mapstructure:"mime"`
	Any   []CategoryFilter `yaml:"any,omitempty"   mapstructure:"any"`
	All   []CategoryFilter `yaml:"all,omitempty"   mapstructure:"all"`
	Not   []CategoryFilter `yaml:"not,omitempty"   mapstructure:"not"`
}

// IsZero lets yaml.v3 omit an empty CategoryFilter when the parent field has omitempty.
func (f CategoryFilter) IsZero() bool {
	return f.Match == nil && f.Age == nil && f.Size == nil && f.Mime == "" &&
		len(f.Any) == 0 && len(f.All) == 0 && len(f.Not) == 0
}

// MatchFilter constrains by filename: one of literal, regex, or glob (mutually exclusive).
type MatchFilter struct {
	Literal       string         `yaml:"literal,omitempty"        mapstructure:"literal"`
	Regex         string         `yaml:"regex,omitempty"          mapstructure:"regex"`
	Glob          string         `yaml:"glob,omitempty"           mapstructure:"glob"`
	CaseSensitive bool           `yaml:"case-sensitive,omitempty" mapstructure:"case-sensitive"`
	CompiledRegex *regexp.Regexp `yaml:"-"                        mapstructure:"-"`
}

// AgeFilter constrains by modification time.
type AgeFilter struct {
	Min time.Duration `yaml:"min,omitempty" mapstructure:"min"`
	Max time.Duration `yaml:"max,omitempty" mapstructure:"max"`
}

// SizeFilter constrains by file size.
type SizeFilter struct {
	Min      string `yaml:"min,omitempty" mapstructure:"min"`
	Max      string `yaml:"max,omitempty" mapstructure:"max"`
	MinBytes int64  `yaml:"-"             mapstructure:"-"`
	MaxBytes int64  `yaml:"-"             mapstructure:"-"`
}

// CategoryHooks holds optional before/after hooks for a category.
type CategoryHooks struct {
	Before *CategoryHook `yaml:"before,omitempty" mapstructure:"before"`
	After  *CategoryHook `yaml:"after,omitempty"  mapstructure:"after"`
}

// CategoryHook defines a list of shell commands to run at a lifecycle point.
type CategoryHook struct {
	Shell     string   `yaml:"shell,omitempty" mapstructure:"shell"`
	OnFailure string   `yaml:"on-failure"      mapstructure:"on-failure"`
	Run       []string `yaml:"run"             mapstructure:"run"`
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
			Description: "File operation to perform. 'move' removes the source; 'copy' keeps it; 'symlink' links it; 'archive' packs the whole category into one compressed file (requires the archive block).",
			OneOf:       []string{"move", "copy", "symlink", "archive"},
			Default:     "move",
			Example:     "action: move",
		}},
		"archive": {FieldMeta: editor.FieldMeta{
			Description: "Archiving options. Required when action is 'archive'. Packs all matched files of the category into one zip/tar.gz at the destination path.",
		}},
		"rename": {FieldMeta: editor.FieldMeta{
			Description: "Token pattern for the destination filename. It becomes the whole filename, so include {ext} to keep the extension (omit it and the file is written without one). Leave empty to keep the original name.",
			Formats:     []editor.Format{FormatRenamePattern},
			Example:     "rename: \"{year}-{month}-{day}_{name}.{ext}\"\n\n# Full filename — include {ext} to keep the extension.\n# {name}  filename without extension\n# {year}, {month}, {day}, {hour}, {minute}, {second}\n# {seq}   auto-incrementing counter\n# {sha256:N}  first N hex chars of SHA-256",
		}},
	}
}

func (ArchiveConfig) Metadata() map[string]*metadata.Node {
	return map[string]*metadata.Node{
		"format": {FieldMeta: editor.FieldMeta{
			Description: "Archive container/compression: 'zip' (universal) or 'tar.gz'. Required — no default.",
			Required:    true,
			OneOf:       []string{"zip", "tar.gz"},
			Example:     "format: zip",
		}},
		"name": {FieldMeta: editor.FieldMeta{
			Description: "Archive filename base (extension is added automatically). Supports category/date/system tokens only: {category}, {date}, {timestamp}, {hostname}, {username}, {os}. Empty uses the category name.",
			Example:     "name: \"{category}_{date}\"",
		}},
		"compression": {FieldMeta: editor.FieldMeta{
			Description: "Compression effort: 'none', 'fast', or 'best'.",
			OneOf:       []string{"none", "fast", "best"},
			Default:     "best",
			Example:     "compression: best",
		}},
		"keep-source": {FieldMeta: editor.FieldMeta{
			Description: "Keep the original files after archiving. Defaults to true; set false to delete sources after a successful write.",
			Default:     "true",
			Example:     "keep-source: true",
		}},
		"flatten": {FieldMeta: editor.FieldMeta{
			Description: "Put every file at the archive root. Defaults to false, which preserves each file's sub-path relative to the source directory (relevant with recursive scans).",
			Default:     "false",
			Example:     "flatten: false",
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
		"mime": {FieldMeta: editor.FieldMeta{
			Description: "Match by the file's real MIME type (magic bytes), as a glob against the detected type. Examples: \"image/*\", \"application/pdf\". Reads the file content; combine with extensions: [all] to match by real type.",
			Example:     "mime: \"image/*\"",
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
