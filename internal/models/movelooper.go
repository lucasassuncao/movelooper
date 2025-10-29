// Package models defines the data structures and functions related to application configuration.
package models

import (
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

// Movelooper is a struct that holds the logger, viper, flags and category config
type Movelooper struct {
	Logger         *pterm.Logger
	Viper          *viper.Viper
	Flags          *Flags
	CategoryConfig []*CategoryConfig
}

// CategoryConfig is a struct that holds the category configuration
type CategoryConfig struct {
	CategoryName string   `mapstructure:"name"`
	Extensions   []string `mapstructure:"extensions"`
	Source       string   `mapstructure:"source"`
	Destination  string   `mapstructure:"destination"`
}
