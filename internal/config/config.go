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
		f := &cat.Source.Filter

		if len(cat.Source.Extensions) == 0 {
			return nil, fmt.Errorf("category %q: source.extensions are required", cat.Name)
		}

		if f.Regex != "" && f.Glob != "" {
			return nil, fmt.Errorf("category %q: source.filter regex and glob are mutually exclusive; use only one", cat.Name)
		}

		if f.Regex != "" {
			compiled, err := regexp.Compile(f.Regex)
			if err != nil {
				return nil, fmt.Errorf("invalid regex in category %q: %w", cat.Name, err)
			}
			f.CompiledRegex = compiled
		}

		if f.Glob != "" {
			if err := helper.ValidateGlob(f.Glob); err != nil {
				return nil, fmt.Errorf("category %q: %w", cat.Name, err)
			}
		}

		if f.MinSize != "" {
			bytes, err := helper.ParseSize(f.MinSize)
			if err != nil {
				return nil, fmt.Errorf("category %q: invalid min-size %q: %w", cat.Name, f.MinSize, err)
			}
			f.MinSizeBytes = bytes
		}

		if f.MaxSize != "" {
			bytes, err := helper.ParseSize(f.MaxSize)
			if err != nil {
				return nil, fmt.Errorf("category %q: invalid max-size %q: %w", cat.Name, f.MaxSize, err)
			}
			f.MaxSizeBytes = bytes
		}

		if f.MinSize != "" && f.MaxSize != "" && f.MinSizeBytes > f.MaxSizeBytes {
			return nil, fmt.Errorf("category %q: min-size (%s) must be less than max-size (%s)", cat.Name, f.MinSize, f.MaxSize)
		}

		if f.MinAge != 0 && f.MaxAge != 0 && f.MinAge > f.MaxAge {
			return nil, fmt.Errorf("category %q: min-age (%s) must be less than max-age (%s)", cat.Name, f.MinAge, f.MaxAge)
		}
	}

	return categories, nil
}
