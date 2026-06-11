package cmd

import "github.com/lucasassuncao/yedit/editor"

// MovelooperValidators is the rule set enforced by the edit command at
// validate/save time.
//
// Per-field constraints (required, allowed values, ranges, counts,
// uniqueness) are declared once in the hint tree (edit_hints.go) and enforced
// by the FromMetadata family — hints are the single source of field metadata.
// Only cross-field rules, which cannot live in per-field metadata, are
// declared here explicitly.
var MovelooperValidators = []editor.Validator{
	// Enforce everything the metadata declares.
	editor.RequiredFromMetadata(),
	editor.OneOfFromMetadata(),
	editor.RangeFromMetadata(),
	editor.PatternFromMetadata(),
	editor.CountFromMetadata(),
	editor.UniqueFromMetadata(),
	editor.DeprecatedFromMetadata(),

	// Category names must be unique across the list (presence comes from the
	// hints; NoDuplicates skips unnamed entries).
	editor.NoDuplicates("categories", "name"),
	// any/all are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "any", "all"),
	// regex/glob are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "regex", "glob"),

	// min/max pairs must be ordered; mirrors the per-level enumeration used by
	// the MutuallyExclusiveNested rules above.
	editor.CrossFieldOrdered("categories.source.filter.min-age", "categories.source.filter.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.min-size", "categories.source.filter.max-size"),
	editor.CrossFieldOrdered("categories.source.filter.any.min-age", "categories.source.filter.any.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.any.min-size", "categories.source.filter.any.max-size"),
	editor.CrossFieldOrdered("categories.source.filter.all.min-age", "categories.source.filter.all.max-age"),
	editor.CrossFieldOrdered("categories.source.filter.all.min-size", "categories.source.filter.all.max-size"),
}
