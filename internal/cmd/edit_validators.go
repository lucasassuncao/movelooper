package cmd

import (
	"fmt"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
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

	// any and all are mutually exclusive with each other and with leaf fields
	// (match/age/size). not is a modifier and may coexist with any/all.
	// Four validators cover all nesting depths.
	editor.MutuallyExclusiveGroupsNested("categories.source.filter", []string{"any"}, []string{"all"}, []string{"match", "age", "size"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.any", []string{"any"}, []string{"all"}, []string{"match", "age", "size"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.all", []string{"any"}, []string{"all"}, []string{"match", "age", "size"}),
	editor.MutuallyExclusiveGroupsNested("categories.source.filter.not", []string{"any"}, []string{"all"}, []string{"match", "age", "size"}),

	// age and size min/max pairs must be ordered at any nesting depth.
	editor.CrossFieldOrderedNested("categories.source.filter.age", "min", "max"),
	editor.CrossFieldOrderedNested("categories.source.filter.size", "min", "max"),

	// filter nesting (any/all/not) cannot exceed config.MaxFilterNestingDepth.
	// Reuses config.FilterDepthOK so the limit is enforced here in the TUI and
	// not just on the next `movelooper` run.
	editor.ValidatorFunc(func(in editor.ValidationInput) []editor.Violation {
		var doc struct {
			Categories []struct {
				Source struct {
					Filter models.CategoryFilter `yaml:"filter"`
				} `yaml:"source"`
			} `yaml:"categories"`
		}
		if err := yaml.Unmarshal(in.Raw, &doc); err != nil {
			return nil
		}
		var errs []editor.Violation
		for i, c := range doc.Categories {
			if !config.FilterDepthOK(&c.Source.Filter, config.MaxFilterNestingDepth, 0) {
				errs = append(errs, editor.Violation{
					Path:    fmt.Sprintf("categories[%d].source.filter", i),
					Message: fmt.Sprintf("nesting exceeds maximum depth of %d", config.MaxFilterNestingDepth),
				})
			}
		}
		return errs
	}),

	// Custom validation to check that logging.file is required when output is "file" or "both".
	editor.ValidatorFunc(func(in editor.ValidationInput) []editor.Violation {
		var doc struct {
			Configuration struct {
				Logging struct {
					Output string `yaml:"output"`
					File   string `yaml:"file"`
				} `yaml:"logging"`
			} `yaml:"configuration"`
		}
		if err := yaml.Unmarshal(in.Raw, &doc); err != nil {
			return nil
		}
		log := doc.Configuration.Logging
		if log.Output != "file" && log.Output != "both" {
			return nil
		}
		if log.File != "" {
			return nil
		}
		return []editor.Violation{{
			Path:    "configuration.logging.file",
			Message: fmt.Sprintf("required when output is %q", log.Output),
		}}
	}),

	// Validate rename and organize-by templates against the known token set.
	// Also enforces that {seq}/{seq:N} is not used in organize-by (rename only).
	editor.ValidatorFunc(func(in editor.ValidationInput) []editor.Violation {
		var doc struct {
			Categories []struct {
				Destination struct {
					Rename     string `yaml:"rename"`
					OrganizeBy string `yaml:"organize-by"`
				} `yaml:"destination"`
			} `yaml:"categories"`
		}
		if err := yaml.Unmarshal(in.Raw, &doc); err != nil {
			return nil
		}
		var errs []editor.Violation
		for i, c := range doc.Categories {
			if err := tokens.ValidateTemplate(c.Destination.Rename); err != nil {
				errs = append(errs, editor.Violation{
					Path:    fmt.Sprintf("categories[%d].destination.rename", i),
					Message: err.Error(),
				})
			}
			if err := tokens.ValidateTemplate(c.Destination.OrganizeBy); err != nil {
				errs = append(errs, editor.Violation{
					Path:    fmt.Sprintf("categories[%d].destination.organize-by", i),
					Message: err.Error(),
				})
			} else if tokens.ContainsSeqToken(c.Destination.OrganizeBy) {
				errs = append(errs, editor.Violation{
					Path:    fmt.Sprintf("categories[%d].destination.organize-by", i),
					Message: "{seq} is not valid in organize-by; use it in rename only",
				})
			}
		}
		return errs
	}),
}
