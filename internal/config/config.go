package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// ErrConfigNotFound is returned by InitConfig when the config file cannot be located.
var ErrConfigNotFound = fmt.Errorf("config file not found")

// InitConfig reads the YAML file at path, resolves any import: entries,
// and loads the merged document into k. Returns ErrConfigNotFound when
// the file does not exist, or a descriptive error for any other failure.
func InitConfig(k *koanf.Koanf, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%w: %w", ErrConfigNotFound, err)
	}

	merged, err := ResolveImports(path)
	if err != nil {
		return err
	}

	return k.Load(rawbytes.Provider(merged), yaml.Parser())
}

// UnmarshalConfig reads categories from k, validates them, and pre-compiles
// regex patterns. Returns an error if any category is misconfigured.
func UnmarshalConfig(k *koanf.Koanf) ([]*models.Category, error) {
	var categories []*models.Category
	if err := k.UnmarshalWithConf("categories", &categories, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return nil, fmt.Errorf("unable to decode categories: %w", err)
	}

	for _, cat := range categories {
		if err := validateCategory(cat); err != nil {
			return nil, err
		}
	}

	return categories, nil
}

// validActions is the set of accepted values for destination.action.
var validActions = map[string]bool{
	"":        true, // empty = default (move)
	"move":    true,
	"copy":    true,
	"symlink": true,
}

// validateCategory validates a single category and pre-compiles its filter.
func validateCategory(cat *models.Category) error {
	if len(cat.Source.Extensions) == 0 {
		return fmt.Errorf("category %q: source.extensions are required", cat.Name)
	}

	if !validActions[cat.Destination.Action] {
		return fmt.Errorf("category %q: invalid action %q - must be move, copy, or symlink", cat.Name, cat.Destination.Action)
	}

	if cat.Destination.Rename != "" {
		if err := helper.ValidateTemplate(cat.Destination.Rename); err != nil {
			return fmt.Errorf("category %q: invalid rename template: %w", cat.Name, err)
		}
	}

	if cat.Destination.OrganizeBy != "" {
		if err := helper.ValidateTemplate(cat.Destination.OrganizeBy); err != nil {
			return fmt.Errorf("category %q: invalid organize-by template: %w", cat.Name, err)
		}
		if containsSeqToken(cat.Destination.OrganizeBy) {
			return fmt.Errorf("category %q: {seq} is not valid in organize-by; use it in rename only", cat.Name)
		}
	}

	if err := validateHooks(cat.Name, cat.Hooks); err != nil {
		return err
	}

	return validateFilter(cat.Name, &cat.Source.Filter)
}

// validateHooks validates both before and after hooks for a category.
func validateHooks(catName string, hooks *models.CategoryHooks) error {
	if hooks == nil {
		return nil
	}
	if hooks.Before != nil {
		if err := validateHook(catName, "before", hooks.Before); err != nil {
			return err
		}
	}
	if hooks.After != nil {
		if err := validateHook(catName, "after", hooks.After); err != nil {
			return err
		}
	}
	return nil
}

// validateHook validates a single CategoryHook.
func validateHook(catName, position string, hook *models.CategoryHook) error {
	if len(hook.Run) == 0 {
		return fmt.Errorf("category %q: hooks.%s.run must not be empty", catName, position)
	}
	if hook.OnFailure != "abort" && hook.OnFailure != "warn" {
		return fmt.Errorf("category %q: hooks.%s.on-failure must be \"abort\" or \"warn\"", catName, position)
	}
	return nil
}

// hasDirectFilterFields reports whether f has any direct filter fields set.
func hasDirectFilterFields(f *models.CategoryFilter) bool {
	return f.Regex != "" || f.Glob != "" ||
		len(f.Include) > 0 || len(f.Ignore) > 0 ||
		f.MinAge != 0 || f.MaxAge != 0 ||
		f.MinSize != "" || f.MaxSize != ""
}

// validateFilter validates a filter node recursively.
// Nodes with any/all are validated as composite nodes; all others as leaves.
func validateFilter(catName string, f *models.CategoryFilter) error {
	hasAny := len(f.Any) > 0
	hasAll := len(f.All) > 0

	if f.Any != nil && !hasAny {
		return fmt.Errorf("category %q: filter 'any' must have at least one entry", catName)
	}
	if f.All != nil && !hasAll {
		return fmt.Errorf("category %q: filter 'all' must have at least one entry", catName)
	}
	if hasAny && hasAll {
		return fmt.Errorf("category %q: filter cannot have both 'any' and 'all' at the same level", catName)
	}
	if (hasAny || hasAll) && hasDirectFilterFields(f) {
		return fmt.Errorf("category %q: filter cannot mix 'any'/'all' with direct fields", catName)
	}

	if hasAny {
		for i := range f.Any {
			if err := validateFilter(catName, &f.Any[i]); err != nil {
				return err
			}
		}
		return nil
	}
	if hasAll {
		for i := range f.All {
			if err := validateFilter(catName, &f.All[i]); err != nil {
				return err
			}
		}
		return nil
	}

	return validateLeafFilter(catName, f)
}

// seqInTemplate matches any {seq} or {seq:N} token in a template string.
var seqInTemplate = regexp.MustCompile(`\{seq(?::\d+)?\}`)

// containsSeqToken reports whether template contains a {seq} or {seq:N} token.
func containsSeqToken(template string) bool {
	return seqInTemplate.MatchString(template)
}

// validateLeafFilter validates a plain (non-composite) filter node.
func validateLeafFilter(catName string, f *models.CategoryFilter) error {
	if f.Regex != "" && f.Glob != "" {
		return fmt.Errorf("category %q: source.filter regex and glob are mutually exclusive; use only one", catName)
	}

	if err := compileRegex(catName, f); err != nil {
		return err
	}

	if f.Glob != "" {
		if err := helper.ValidateGlob(f.Glob); err != nil {
			return fmt.Errorf("category %q: %w", catName, err)
		}
	}

	for _, p := range f.Include {
		if err := helper.ValidateGlob(p); err != nil {
			return fmt.Errorf("category %q: invalid include pattern: %w", catName, err)
		}
	}

	return validateSizeAndAge(catName, f)
}

// compileRegex compiles f.Regex into f.CompiledRegex, adding (?i) when not case-sensitive.
func compileRegex(catName string, f *models.CategoryFilter) error {
	if f.Regex == "" {
		return nil
	}
	pattern := f.Regex
	if !f.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex in category %q: %w", catName, err)
	}
	f.CompiledRegex = compiled
	return nil
}

// validateSizeAndAge parses size strings and checks that min <= max for both size and age.
func validateSizeAndAge(catName string, f *models.CategoryFilter) error {
	if f.MinSize != "" {
		b, err := helper.ParseSize(f.MinSize)
		if err != nil {
			return fmt.Errorf("category %q: invalid min-size %q: %w", catName, f.MinSize, err)
		}
		f.MinSizeBytes = b
	}

	if f.MaxSize != "" {
		b, err := helper.ParseSize(f.MaxSize)
		if err != nil {
			return fmt.Errorf("category %q: invalid max-size %q: %w", catName, f.MaxSize, err)
		}
		f.MaxSizeBytes = b
	}

	if f.MinSize != "" && f.MaxSize != "" && f.MinSizeBytes > f.MaxSizeBytes {
		return fmt.Errorf("category %q: min-size (%s) must be less than max-size (%s)", catName, f.MinSize, f.MaxSize)
	}

	if f.MinAge != 0 && f.MaxAge != 0 && f.MinAge > f.MaxAge {
		return fmt.Errorf("category %q: min-age (%s) must be less than max-age (%s)", catName, f.MinAge, f.MaxAge)
	}

	return nil
}
