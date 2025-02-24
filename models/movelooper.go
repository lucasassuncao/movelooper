package models

import (
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

type Movelooper struct {
	Logger      *pterm.Logger
	Viper       *viper.Viper
	Flags       *PersistentFlags
	MediaConfig *MediaConfig
}

type MediaConfig struct {
	AllCategories []string
	Category      string
	Extensions    []string
	Source        string
	Destination   string
}
