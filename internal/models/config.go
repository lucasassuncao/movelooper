package models

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Configuration Configuration `yaml:"configuration"`
	Categories    []Category    `yaml:"categories"`
}

type Configuration struct {
	Output     string `yaml:"output"`
	LogFile    string `yaml:"log-file"`
	LogLevel   string `yaml:"log-level"`
	ShowCaller bool   `yaml:"show-caller"`
}

type Category struct {
	Name        string   `yaml:"name"`
	Extensions  []string `yaml:"extensions"`
	Source      string   `yaml:"source"`
	Destination string   `yaml:"destination"`
}

func NewConfig(path string) error {
	baseConfig := Config{
		Configuration: Configuration{
			Output:     "",
			LogFile:    "",
			LogLevel:   "",
			ShowCaller: false,
		},
		Categories: []Category{
			{
				Name:        "foo",
				Extensions:  []string{"fizz", "buzz"},
				Source:      "",
				Destination: "",
			},
			{
				Name:        "bar",
				Extensions:  []string{"ping", "pong"},
				Source:      "",
				Destination: "",
			},
			{
				Name:        "yin",
				Extensions:  []string{"zip", "zap"},
				Source:      "",
				Destination: "",
			},
			{
				Name:        "yang",
				Extensions:  []string{"beep", "boop"},
				Source:      "",
				Destination: "",
			},
		},
	}

	data, err := yaml.Marshal(&baseConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(path, "movelooper.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to generate base config file: %w", err)
	}

	return nil
}
