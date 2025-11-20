package models

import (
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

// Movelooper holds the app dependencies and runtime state
type Movelooper struct {
	Logger     *pterm.Logger
	Viper      *viper.Viper
	Flags      *Flags
	Categories []*Category
}

// Config represents the complete structure of the movelooper.yaml file
type Config struct {
	Configuration Configuration `yaml:"configuration" mapstructure:"configuration"`
	Categories    []Category    `yaml:"categories" mapstructure:"categories"`
}

// Configuration holds the general settings for Movelooper
type Configuration struct {
	Output     string        `yaml:"output" mapstructure:"output"`
	LogFile    string        `yaml:"log-file" mapstructure:"log-file"`
	LogLevel   string        `yaml:"log-level" mapstructure:"log-level"`
	ShowCaller bool          `yaml:"show-caller" mapstructure:"show-caller"`
	WatchDelay time.Duration `yaml:"watch_delay" mapstructure:"watch_delay"` // Adicionado para o modo Watch
}

// Category represents a file category with its properties
type Category struct {
	Name             string   `yaml:"name" mapstructure:"name"`
	Extensions       []string `yaml:"extensions" mapstructure:"extensions"`
	Source           string   `yaml:"source" mapstructure:"source"`
	Destination      string   `yaml:"destination" mapstructure:"destination"`
	ConflictStrategy string   `yaml:"conflict_strategy" mapstructure:"conflict_strategy"`
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
