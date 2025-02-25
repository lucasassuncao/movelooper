package config

import (
	"fmt"
	"log"
	"movelooper/models"

	"github.com/spf13/viper"
)

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

// UnmarshalConfig unmarshals the config file into a struct
func UnmarshalConfig(m *models.Movelooper) []*models.MediaConfig {
	var categories []*models.MediaConfig
	if err := m.Viper.UnmarshalKey("categories", &categories); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	return categories
}
