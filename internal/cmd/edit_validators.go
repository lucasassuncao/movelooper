package cmd

import "github.com/lucasassuncao/yedit/editor"

var MovelooperValidators = []editor.Validator{
	// Category names must be unique across the list.
	editor.NoDuplicates("categories", "name"),
	// Every category needs a name (NoDuplicates skips unnamed entries).
	editor.Required("categories.name"),
	// any/all are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "any", "all"),
	// regex/glob are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "regex", "glob"),

	// Enum values enforced at edit time instead of failing at config load.
	editor.ValueOneOf("configuration.output", "console", "file", "both"),
	editor.ValueOneOf("configuration.log-level", "trace", "debug", "info", "warn", "error", "fatal"),
	editor.ValueOneOf("categories.destination.action", "move", "copy", "symlink"),
	editor.ValueOneOf("categories.destination.conflict-strategy",
		"rename", "hash_check", "overwrite", "skip", "newest", "oldest", "larger", "smaller"),
	editor.ValueOneOf("categories.hooks.before.on-failure", "abort", "warn"),
	editor.ValueOneOf("categories.hooks.after.on-failure", "abort", "warn"),

	// min/max pairs must be ordered; mirrors the per-level enumeration used by
	// the MutuallyExclusiveNested rules above.
	editor.CrossFieldOrdered("categories.source.filter.min-age", "categories.source.filter.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.min-size", "categories.source.filter.max-size"),
	editor.CrossFieldOrdered("categories.source.filter.any.min-age", "categories.source.filter.any.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.any.min-size", "categories.source.filter.any.max-size"),
	editor.CrossFieldOrdered("categories.source.filter.all.min-age", "categories.source.filter.all.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.all.min-size", "categories.source.filter.all.max-size"),
}
