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

// UnmarshalConfig unmarshals the config file into a struct and pre-compiles regex patterns.
// Returns an error if any category has an invalid regex.
func UnmarshalConfig(m *models.Movelooper) ([]*models.Category, error) {
	var categories []*models.Category
	if err := m.Viper.UnmarshalKey("categories", &categories); err != nil {
		return nil, fmt.Errorf("unable to decode categories: %w", err)
	}

	for _, cat := range categories {
		if len(cat.Extensions) == 0 {
			return nil, fmt.Errorf("category %q: extensions are required", cat.Name)
		}

		if cat.Regex != "" && cat.Glob != "" {
			return nil, fmt.Errorf("category %q: regex and glob are mutually exclusive; use only one", cat.Name)
		}

		if cat.Regex != "" {
			compiled, err := regexp.Compile(cat.Regex)
			if err != nil {
				return nil, fmt.Errorf("invalid regex in category %q: %w", cat.Name, err)
			}
			cat.CompiledRegex = compiled
		}

		if cat.Glob != "" {
			if err := helper.ValidateGlob(cat.Glob); err != nil {
				return nil, fmt.Errorf("category %q: %w", cat.Name, err)
			}
		}

		if cat.MinSize != "" {
			bytes, err := helper.ParseSize(cat.MinSize)
			if err != nil {
				return nil, fmt.Errorf("category %q: invalid min-size %q: %w", cat.Name, cat.MinSize, err)
			}
			cat.MinSizeBytes = bytes
		}
	}

	return categories, nil
}
