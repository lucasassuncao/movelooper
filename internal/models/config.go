package models

import "time"

// Config represents the complete structure of the movelooper.yaml file
type Config struct {
	Configuration Configuration `yaml:"configuration" mapstructure:"configuration"`
	Categories    []Category    `yaml:"categories" mapstructure:"categories"`
}

// Configuration holds the general settings for Movelooper
type Configuration struct {
	Output       string        `yaml:"output" mapstructure:"output"`
	LogFile      string        `yaml:"log-file" mapstructure:"log-file"`
	LogLevel     string        `yaml:"log-level" mapstructure:"log-level"`
	ShowCaller   bool          `yaml:"show-caller" mapstructure:"show-caller"`
	WatchDelay   time.Duration `yaml:"watch-delay" mapstructure:"watch-delay"`
	HistoryLimit int           `yaml:"history-limit" mapstructure:"history-limit"`
}
