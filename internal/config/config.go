package config

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/viper"
)

// ViperOptions is a function that takes a viper instance and applies options to it
type ViperOptions func(*viper.Viper)

// ErrConfigNotFound is returned by InitConfig when the base config file cannot be located.
var ErrConfigNotFound = fmt.Errorf("config file not found")

// InitConfig initializes Viper to read from movelooper.yaml.
// After locating the config file it resolves any top-level `import:` entries,
// merges all imported categories, and re-feeds the merged document into Viper.
// Returns ErrConfigNotFound (unwrappable) when the base file does not exist,
// or a descriptive error for any other failure (e.g. a missing imported file).
func InitConfig(v *viper.Viper, options ...ViperOptions) error {
	applyOptions(v, options...)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("%w: %w", ErrConfigNotFound, err)
	}

	merged, err := ResolveImports(v.ConfigFileUsed())
	if err != nil {
		return err
	}

	return v.ReadConfig(bytes.NewReader(merged))
}

// applyOptions applies the options to the viper instance
func applyOptions(v *viper.Viper, options ...ViperOptions) {
	for _, option := range options {
		option(v)
	}
}

// WithConfigName sets the name of the config file
func WithConfigName(name string) ViperOptions {
	return func(v *viper.Viper) {
		v.SetConfigName(name)
	}
}

// WithConfigType sets the type of the config file
func WithConfigType(configType string) ViperOptions {
	return func(v *viper.Viper) {
		v.SetConfigType(configType)
	}
}

// WithConfigPath sets the path of the config file
func WithConfigPath(path string) ViperOptions {
	return func(v *viper.Viper) {
		v.AddConfigPath(path)
	}
}

// UnmarshalConfig reads categories from v, validates them, and pre-compiles
// regex patterns. Returns an error if any category is misconfigured.
func UnmarshalConfig(v *viper.Viper) ([]*models.Category, error) {
	var categories []*models.Category
	if err := v.UnmarshalKey("categories", &categories); err != nil {
		return nil, fmt.Errorf("unable to decode categories: %w", err)
	}

	for _, cat := range categories {
		if err := validateCategory(cat); err != nil {
			return nil, err
		}
	}

	return categories, nil
}

// validateCategory validates a single category and pre-compiles its filter.
func validateCategory(cat *models.Category) error {
	if len(cat.Source.Extensions) == 0 {
		return fmt.Errorf("category %q: source.extensions are required", cat.Name)
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
