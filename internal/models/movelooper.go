package models

import (
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

// Movelooper is a struct that holds the logger, viper, flags and media config
type Movelooper struct {
	Logger      *pterm.Logger
	Viper       *viper.Viper
	Flags       *PersistentFlags
	MediaConfig []*MediaConfig
}

// PersistentFlags is a struct that holds the persistent flags that are used by the CLI
type MediaConfig struct {
	CategoryName string   `mapstructure:"name"`
	Extensions   []string `mapstructure:"extensions"`
	Source       string   `mapstructure:"source"`
	Destination  string   `mapstructure:"destination"`
}
