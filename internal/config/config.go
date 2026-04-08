package config

import (
	"fmt"
	"regexp"

	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/viper"
)

// ViperOptions is a function that takes a viper instance and applies options to it
type ViperOptions func(*viper.Viper)

// InitConfig initializes Viper to read from movelooper.yaml
func InitConfig(v *viper.Viper, options ...ViperOptions) error {
	applyOptions(v, options...)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("could not read config: %w", err)
	}
	return nil
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
		if len(cat.Extensions) == 0 {
			return nil, fmt.Errorf("category %q: extensions are required", cat.Name)
		}

		if cat.Filter.Regex != "" && cat.Filter.Glob != "" {
			return nil, fmt.Errorf("category %q: regex and glob are mutually exclusive; use only one", cat.Name)
		}

		if cat.Filter.Regex != "" {
			compiled, err := regexp.Compile(cat.Filter.Regex)
			if err != nil {
				return nil, fmt.Errorf("invalid regex in category %q: %w", cat.Name, err)
			}
			cat.Filter.CompiledRegex = compiled
		}

		if cat.Filter.Glob != "" {
			if err := helper.ValidateGlob(cat.Filter.Glob); err != nil {
				return nil, fmt.Errorf("category %q: %w", cat.Name, err)
			}
		}

		if cat.Filter.MinSize != "" {
			bytes, err := helper.ParseSize(cat.Filter.MinSize)
			if err != nil {
				return nil, fmt.Errorf("category %q: invalid min-size %q: %w", cat.Name, cat.Filter.MinSize, err)
			}
			cat.Filter.MinSizeBytes = bytes
		}

		if cat.Filter.MaxSize != "" {
			bytes, err := helper.ParseSize(cat.Filter.MaxSize)
			if err != nil {
				return nil, fmt.Errorf("category %q: invalid max-size %q: %w", cat.Name, cat.Filter.MaxSize, err)
			}
			cat.Filter.MaxSizeBytes = bytes
		}

		if cat.Filter.MinSize != "" && cat.Filter.MaxSize != "" && cat.Filter.MinSizeBytes > cat.Filter.MaxSizeBytes {
			return nil, fmt.Errorf("category %q: min-size (%s) must be less than max-size (%s)", cat.Name, cat.Filter.MinSize, cat.Filter.MaxSize)
		}

		if cat.Filter.MinAge != 0 && cat.Filter.MaxAge != 0 && cat.Filter.MinAge > cat.Filter.MaxAge {
			return nil, fmt.Errorf("category %q: min-age (%s) must be less than max-age (%s)", cat.Name, cat.Filter.MinAge, cat.Filter.MaxAge)
		}
	}

	return categories, nil
}
