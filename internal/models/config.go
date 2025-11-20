// Package models defines the data structures and functions related to application configuration.
package models

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/pterm/pterm"
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
	Name             string   `yaml:"name"`
	Extensions       []string `yaml:"extensions"`
	Source           string   `yaml:"source"`
	Destination      string   `yaml:"destination"`
	ConflictStrategy string   `yaml:"conflict_strategy"`
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
	logFile, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Specify the log file").WithDefaultValue("C:\\logs\\movelooper.log").Show()

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
