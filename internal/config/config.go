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
		return fmt.Errorf("category %q: invalid action %q — must be move, copy, or symlink", cat.Name, cat.Destination.Action)
	}

	if cat.Destination.Rename != "" {
		if err := helper.ValidateTemplate(cat.Destination.Rename); err != nil {
			return fmt.Errorf("category %q: invalid rename template: %w", cat.Name, err)
		}
	}

	return validateFilter(cat.Name, &cat.Source.Filter)
}

// validateFilter validates and pre-compiles all filter fields for a category.
func validateFilter(catName string, f *models.CategoryFilter) error {
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
