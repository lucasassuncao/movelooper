package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/hints"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
)

// filterChildren is defined once and shared via pointers in any/all nodes,
// so CategoryFilter's self-referential structure resolves at any depth:
//
//	filter.any.regex           → filterChildren["regex"]
//	filter.any.all.regex       → filterChildren["all"].Children["regex"] (same map)
//	filter.any.any.any.ignore  → same
var filterChildren map[string]*hints.HintNode

func init() {
	anyNode := &hints.HintNode{
		FieldMeta: editor.FieldMeta{
			Description: "OR logic: file must match at least one sub-filter.",
			Example:     "any:\n  - regex: \"^invoice_.*\"\n  - glob: \"receipt_*\"",
		},
	}
	allNode := &hints.HintNode{
		FieldMeta: editor.FieldMeta{
			Description: "AND logic: file must match all sub-filters simultaneously.",
			Example:     "all:\n  - min-size: 100KB\n  - max-age: 168h",
		},
	}
	filterChildren = map[string]*hints.HintNode{
		"regex": {FieldMeta: editor.FieldMeta{
			Description: "RE2 regular expression matched against the filename (without path). Mutually exclusive with glob.",
			Example:     "regex: \"^\\d{4}-\\d{2}-\\d{2}_.*\\.pdf$\"",
		}},
		"glob": {FieldMeta: editor.FieldMeta{
			Description: "Glob pattern matched against the filename (without path). Mutually exclusive with regex.",
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
			Example:     "min-age: 24h",
		}},
		"max-age": {FieldMeta: editor.FieldMeta{
			Description: "Only match files newer than this duration.",
			Example:     "max-age: 720h",
		}},
		"min-size": {FieldMeta: editor.FieldMeta{
			Description: "Only match files at least this large. Accepts human-readable sizes (e.g. 1MB, 500KB).",
			Example:     "min-size: 1MB",
		}},
		"max-size": {FieldMeta: editor.FieldMeta{
			Description: "Only match files no larger than this size.",
			Example:     "max-size: 50GB",
		}},
		"any": anyNode,
		"all": allNode,
	}
	anyNode.Children = filterChildren
	allNode.Children = filterChildren
	initHints()
}

// MovelooperHints is the editor.HintSource for the movelooper schema.
// Built once at startup via BuildFrom, which reflects over models.Config
// to derive the Type field for each node automatically.
// Initialized in init() so filterChildren is fully set before use.
var MovelooperHints editor.HintSource

func initHints() {
	MovelooperHints = hints.BuildFrom(&models.Config{}, map[string]*hints.HintNode{
		"configuration": {
			FieldMeta: editor.FieldMeta{
				Description: "General settings for movelooper: logging output, watch interval, and history size.",
				Required:    true,
			},
			Children: map[string]*hints.HintNode{
				"output": {FieldMeta: editor.FieldMeta{
					Description: "Where log output is written. Use 'both' to write to the console and a file simultaneously.",
					Required:    true,
					OneOf:       []string{"console", "file", "both"},
					Default:     "console",
					Example:     "output: console",
				}},
				"log-file": {FieldMeta: editor.FieldMeta{
					Description: "Path to the log file. Only used when output is 'file' or 'both'. Supports ~ for the home directory.",
					Default:     "~/movelooper.log",
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
					Example:     "watch-delay: 5m",
				}},
				"history-limit": {FieldMeta: editor.FieldMeta{
					Description: "Maximum number of move events kept in the undo history. Older entries are evicted when the limit is reached.",
					Default:     "100",
					Example:     "history-limit: 100",
				}},
			},
		},
		"categories": {
			FieldMeta: editor.FieldMeta{
				Description: "List of file movement rules. Each entry defines a source directory, file filters, a destination, and optional hooks.",
				Required:    true,
			},
			Children: map[string]*hints.HintNode{
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
				"source": {
					FieldMeta: editor.FieldMeta{
						Description: "Source directory configuration: which path to watch, which extensions to include, and how deep to scan.",
						Required:    true,
					},
					Children: map[string]*hints.HintNode{
						"path": {FieldMeta: editor.FieldMeta{
							Description: "Directory to watch for incoming files.",
							Required:    true,
							Example:     "path: ~/Downloads",
						}},
						"extensions": {FieldMeta: editor.FieldMeta{
							Description: "File extensions to match (without the leading dot). Leave empty to match all extensions.",
							Example:     "extensions:\n  - jpg\n  - jpeg\n  - png\n  - gif",
						}},
						"recursive": {FieldMeta: editor.FieldMeta{
							Description: "Whether to scan sub-directories of the source path. Combine with max-depth to limit depth.",
							Default:     "false",
							Example:     "recursive: true",
						}},
						"max-depth": {FieldMeta: editor.FieldMeta{
							Description: "Maximum sub-directory depth when recursive is true. 0 means unlimited.",
							Default:     "0",
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
							Children: filterChildren,
						},
					},
				},
				"destination": {
					FieldMeta: editor.FieldMeta{
						Description: "Destination configuration: where to place matched files, how to name them, and what to do on conflicts.",
						Required:    true,
					},
					Children: map[string]*hints.HintNode{
						"path": {FieldMeta: editor.FieldMeta{
							Description: "Directory where matched files are placed.",
							Required:    true,
							Example:     "path: ~/Pictures/Sorted",
						}},
						"organize-by": {FieldMeta: editor.FieldMeta{
							Description: "Token pattern used to build sub-directories inside the destination path. Leave empty to place all files directly.",
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
							Example:     "rename: \"{year}-{month}-{day}_{name}\"\n\n# Available tokens:\n# {name}  original filename without extension\n# {ext}   original extension\n# {year}, {month}, {day}, {hour}, {min}, {sec}\n# {seq}   auto-incrementing counter\n# {hash}  SHA-256 prefix of the file content",
						}},
					},
				},
				"hooks": {
					FieldMeta: editor.FieldMeta{
						Description: "Optional shell commands to run before and after each file is moved.",
					},
					Children: map[string]*hints.HintNode{
						"before": {
							FieldMeta: editor.FieldMeta{
								Description: "Hook executed before the file operation. If it fails, the move is aborted (unless on-failure is 'continue').",
							},
							Children: hookChildren("before"),
						},
						"after": {
							FieldMeta: editor.FieldMeta{
								Description: "Hook executed after the file operation completes successfully.",
							},
							Children: hookChildren("after"),
						},
					},
				},
			},
		},
	})
}

func hookChildren(phase string) map[string]*hints.HintNode {
	return map[string]*hints.HintNode{
		"shell": {FieldMeta: editor.FieldMeta{
			Description: "Shell interpreter for the " + phase + "-hook commands.",
			Default:     "/bin/sh",
			Example:     "shell: /bin/bash",
		}},
		"on-failure": {FieldMeta: editor.FieldMeta{
			Description: "What to do if a " + phase + "-hook command exits non-zero.",
			OneOf:       []string{"abort", "continue"},
			Default:     "abort",
			Example:     "on-failure: abort",
		}},
		"run": {FieldMeta: editor.FieldMeta{
			Description: "Shell commands executed in order.",
			Example:     "run:\n  - echo \"" + phase + ": {{.Source}}\"",
		}},
	}
}
