package models

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pterm/pterm"
	yaml "gopkg.in/yaml.v2"
)

// Config represents the full configuration of the application
type Config struct {
	Configuration Configuration `yaml:"configuration"`
	Categories    []Category    `yaml:"categories"`
}

// Configuration represents the configuration of the application
type Configuration struct {
	Output     string `yaml:"output"`
	LogFile    string `yaml:"log-file"`
	LogLevel   string `yaml:"log-level"`
	ShowCaller bool   `yaml:"show-caller"`
}

// Category represents a category of files to move
type Category struct {
	Name        string   `yaml:"name"`
	Extensions  []string `yaml:"extensions"`
	Source      string   `yaml:"source"`
	Destination string   `yaml:"destination"`
}

// ConfigOption is a function that modifies the configuration
type ConfigOption func(*Config)

// WithOutput prompts the user to specify the output
func WithOutput() ConfigOption {
	clearScreen()
	output, _ := pterm.DefaultInteractiveSelect.WithOptions([]string{"console", "log", "file", "both"}).WithDefaultText("Specify the output").WithMaxHeight(10).Show()

	return func(c *Config) {
		c.Configuration.Output = output
	}
}

// WithLogFile prompts the user to specify the log file
func WithLogFile() ConfigOption {
	clearScreen()
	logFile, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the log file").WithDefaultValue("C:\\logs\\gopaper.log").Show()

	return func(c *Config) {
		c.Configuration.LogFile = logFile
	}
}

// WithLogLevel prompts the user to specify the log level
func WithLogLevel() ConfigOption {
	clearScreen()
	logLevel, _ := pterm.DefaultInteractiveSelect.WithOptions([]string{"trace", "debug", "info", "warn", "warning", "error", "fatal"}).WithDefaultText("Specify the log level").WithMaxHeight(10).Show()

	return func(c *Config) {
		c.Configuration.LogLevel = logLevel
	}
}

// WithShowCaller prompts the user to show the caller information
func WithShowCaller() ConfigOption {
	clearScreen()
	showCaller, _ := pterm.DefaultInteractiveConfirm.WithDefaultText("Show caller?").Show()

	return func(c *Config) {
		c.Configuration.ShowCaller = showCaller
	}
}

// WithCategory adds a category to the configuration
// The user is prompted to enter the category name, source directory, destination directory, and extensions
func WithCategory() ConfigOption {
	clearScreen()
	var categories []Category

	want, _ := pterm.DefaultInteractiveConfirm.WithDefaultText("Do you want to add categories?").Show()

	if want {
		for {
			clearScreen()
			var extensions []string

			name, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the category name").Show()
			source, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the source directory").Show()
			destination, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the destination directory").Show()

			wantExtensions, _ := pterm.DefaultInteractiveConfirm.WithDefaultText("Do you want to add extensions?").Show()
			if wantExtensions {
				for {
					extension, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the extension").Show()
					extensions = append(extensions, extension)

					addMore, _ := pterm.DefaultInteractiveConfirm.WithDefaultText("Do you want to add another extension?").Show()
					if !addMore {
						break
					}
				}
			}

			categories = append(categories, Category{
				Name:        name,
				Extensions:  extensions,
				Source:      source,
				Destination: destination,
			})

			addMore, _ := pterm.DefaultInteractiveConfirm.WithDefaultText("Do you want to add another category?").Show()
			if !addMore {
				break
			}
		}
	}

	return func(c *Config) {
		c.Categories = append(c.Categories, categories...)
	}
}

// NewConfig generates a base configuration file
func NewConfig(configPath, baseconfigPath string, interactive bool, configOptions ...ConfigOption) error {
	baseConfig := Config{
		Configuration: Configuration{
			Output:     "",
			LogFile:    "",
			LogLevel:   "",
			ShowCaller: false,
		},
		Categories: []Category{},
	}

	if interactive {
		applyConfigOptions(&baseConfig, configOptions)
	}

	if len(baseConfig.Categories) == 0 {
		baseConfig.Categories = append(baseConfig.Categories, Category{
			Name:        "images",
			Source:      "C:\\wallpapers",
			Destination: "C:\\wallpapers\\moved",
			Extensions:  []string{"jpg", "png", "gif"},
		})
	}

	// Serialize the base config to yaml
	data, err := yaml.Marshal(&baseConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize yaml: %w", err)
	}

	// Write the base config to the base config path
	if err := os.WriteFile(filepath.Join(baseconfigPath, "movelooper.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to generate base config file: %w", err)
	}

	// Move the base config to the config path if it doesn't exist
	oldPath := filepath.Join(baseconfigPath, "movelooper.yaml")
	newPath := filepath.Join(configPath, "movelooper.yaml")

	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		os.Rename(oldPath, newPath)
	}

	clearScreen()
	return nil
}

// applyConfigOptions applies the options to the config instance
func applyConfigOptions(c *Config, configOptions []ConfigOption) {
	for _, option := range configOptions {
		option(c)
	}
}

// clearScreen clears the terminal screen based on the operating system
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
