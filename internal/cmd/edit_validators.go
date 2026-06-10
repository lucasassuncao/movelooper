package cmd

import "github.com/lucasassuncao/yedit/editor"

var MovelooperValidators = []editor.Validator{
	// Category names must be unique across the list.
	editor.NoDuplicates("categories", "name"),
	// any/all are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "any", "all"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "any", "all"),
	// regex/glob are mutually exclusive at every nesting level of filter.
	editor.MutuallyExclusiveNested("categories.source.filter", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.any", "regex", "glob"),
	editor.MutuallyExclusiveNested("categories.source.filter.all", "regex", "glob"),
}
