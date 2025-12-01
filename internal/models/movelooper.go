package models

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/huh"
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
	WatchDelay time.Duration `yaml:"watch-delay" mapstructure:"watch-delay"`
}

// Category represents a file category with its properties
type Category struct {
	Name                  string   `yaml:"name" mapstructure:"name"`
	Extensions            []string `yaml:"extensions" mapstructure:"extensions"`
	Regex                 string   `yaml:"regex" mapstructure:"regex"`
	Source                string   `yaml:"source" mapstructure:"source"`
	Destination           string   `yaml:"destination" mapstructure:"destination"`
	ConflictStrategy      string   `yaml:"conflict_strategy" mapstructure:"conflict_strategy"`
	UseExtensionSubfolder bool     `yaml:"use_extension_subfolder" mapstructure:"use_extension_subfolder"`
}

// ConfigOption is a function that modifies the configuration
type ConfigOption func(*Config)

// WithOutput prompts the user to specify the output
func WithOutput() ConfigOption {
	clearScreen()
	var output string
	err := huh.NewSelect[string]().
		Title("Specify the output").
		Options(
			huh.NewOption("Console", "console"),
			huh.NewOption("Log File", "log"),
			huh.NewOption("File", "file"),
			huh.NewOption("Both", "both"),
		).
		Value(&output).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	return func(c *Config) {
		c.Configuration.Output = output
	}
}

// WithLogFile prompts the user to specify the log file
func WithLogFile() ConfigOption {
	clearScreen()
	defaultLog := "movelooper.log"
	if home, err := os.UserHomeDir(); err == nil {
		defaultLog = filepath.Join(home, "movelooper.log")
	}

	var logFile string
	err := huh.NewInput().
		Title("Specify the log file").
		Value(&logFile).
		Placeholder(defaultLog).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	if logFile == "" {
		logFile = defaultLog
	}

	return func(c *Config) {
		c.Configuration.LogFile = logFile
	}
}

// WithLogLevel prompts the user to specify the log level
func WithLogLevel() ConfigOption {
	clearScreen()
	var logLevel string
	err := huh.NewSelect[string]().
		Title("Specify the log level").
		Options(
			huh.NewOption("Trace", "trace"),
			huh.NewOption("Debug", "debug"),
			huh.NewOption("Info", "info"),
			huh.NewOption("Warn", "warn"),
			huh.NewOption("Warning", "warning"),
			huh.NewOption("Error", "error"),
			huh.NewOption("Fatal", "fatal"),
		).
		Value(&logLevel).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	return func(c *Config) {
		c.Configuration.LogLevel = logLevel
	}
}

// WithShowCaller prompts the user to show the caller information
func WithShowCaller() ConfigOption {
	clearScreen()
	var showCaller bool
	err := huh.NewConfirm().
		Title("Show caller?").
		Value(&showCaller).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	return func(c *Config) {
		c.Configuration.ShowCaller = showCaller
	}
}

// WithCategory adds a category to the configuration
// The user is prompted to enter the category name, source directory, destination directory, and extensions
func WithCategory() ConfigOption {
	clearScreen()
	var categories []Category

	var want bool
	err := huh.NewConfirm().
		Title("Do you want to add categories?").
		Value(&want).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	if want {
		for {
			clearScreen()
			var extensions []string
			var name, source, destination, regex string
			var useRegex, useSubfolder bool

			if err := huh.NewInput().Title("Specify the category name").Value(&name).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}
			if err := huh.NewInput().Title("Specify the source directory").Value(&source).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}
			if err := huh.NewInput().Title("Specify the destination directory").Value(&destination).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}

			if err := huh.NewConfirm().Title("Do you want to use Regex for filtering?").Value(&useRegex).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}

			if useRegex {
				if err := huh.NewInput().Title("Specify the Regex pattern").Value(&regex).Run(); err == huh.ErrUserAborted {
					os.Exit(0)
				}
			} else {
				var wantExtensions bool
				if err := huh.NewConfirm().Title("Do you want to add extensions?").Value(&wantExtensions).Run(); err == huh.ErrUserAborted {
					os.Exit(0)
				}

				if wantExtensions {
					for {
						var extension string
						if err := huh.NewInput().Title("Specify the extension").Value(&extension).Run(); err == huh.ErrUserAborted {
							os.Exit(0)
						}
						extensions = append(extensions, extension)

						var addMore bool
						if err := huh.NewConfirm().Title("Do you want to add another extension?").Value(&addMore).Run(); err == huh.ErrUserAborted {
							os.Exit(0)
						}
						if !addMore {
							break
						}
					}
				}
			}

			if err := huh.NewConfirm().Title("Create subfolders for extensions?").Description("If yes, files will be moved to Destination/Extension/File.ext").Value(&useSubfolder).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}

			categories = append(categories, Category{
				Name:                  name,
				Extensions:            extensions,
				Regex:                 regex,
				Source:                source,
				Destination:           destination,
				UseExtensionSubfolder: useSubfolder,
			})

			var addMore bool
			if err := huh.NewConfirm().Title("Do you want to add another category?").Value(&addMore).Run(); err == huh.ErrUserAborted {
				os.Exit(0)
			}
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
