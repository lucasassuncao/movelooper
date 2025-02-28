package models

import (
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

type Movelooper struct {
	Logger      *pterm.Logger
	Viper       *viper.Viper
	Flags       *PersistentFlags
	MediaConfig []*MediaConfig
}

type MediaConfig struct {
	CategoryName string   `mapstructure:"name"`
	Extensions   []string `mapstructure:"extensions"`
	Source       string   `mapstructure:"source"`
	Destination  string   `mapstructure:"destination"`
}
