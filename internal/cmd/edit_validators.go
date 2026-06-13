package cmd

import (
	"fmt"

	"github.com/lucasassuncao/yedit/editor"
	"gopkg.in/yaml.v3"
)

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
	editor.FormatFromMetadata(),
	editor.LengthFromMetadata(),
	editor.NotOneOfFromMetadata(),

	// Category names must be unique across the list (presence comes from the
	// hints; NoDuplicates skips unnamed entries).
	editor.NoDuplicates("categories", "name"),

	// within match blocks, literal/regex/glob are mutually exclusive at any depth.
	// One validator suffices: MutuallyExclusiveNested walks the full subtree.
	editor.MutuallyExclusiveNested("categories.source.filter.match", "literal", "regex", "glob"),

	// any, all, and leaf fields (match/age/size/not) are three mutually exclusive
	// groups at each filter level. Four validators cover all nesting depths.
	editor.MutuallyExclusiveGroupsNested("categories.source.filter", []string{"any"}, []string{"all"}, []string{"match", "age", "size", "not"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.any", []string{"any"}, []string{"all"}, []string{"match", "age", "size", "not"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.all", []string{"any"}, []string{"all"}, []string{"match", "age", "size", "not"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.not", []string{"any"}, []string{"all"}, []string{"match", "age", "size", "not"}),

	// age and size min/max pairs must be ordered at any nesting depth.
	editor.CrossFieldOrderedNested("categories.source.filter.age", "min", "max"),
	editor.CrossFieldOrderedNested("categories.source.filter.size", "min", "max"),

	// Custom validation to check that log-file is required when output is "file" or "both".
	editor.ValidatorFunc(func(in editor.ValidationInput) []editor.Violation {
		var doc struct {
			Configuration struct {
				Output  string `yaml:"output"`
				LogFile string `yaml:"log-file"`
			} `yaml:"configuration"`
		}
		if err := yaml.Unmarshal(in.Raw, &doc); err != nil {
			return nil
		}
		cfg := doc.Configuration
		if cfg.Output != "file" && cfg.Output != "both" {
			return nil
		}
		if cfg.LogFile != "" {
			return nil
		}
		return []editor.Violation{{
			Path:    "configuration.log-file",
			Message: fmt.Sprintf("required when output is %q", cfg.Output),
		}}
	}),
}
