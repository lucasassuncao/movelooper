package models

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/terminal"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

// Movelooper holds the app dependencies and runtime state
type Movelooper struct {
	Logger     *pterm.Logger
	Viper      *viper.Viper
	Flags      *Flags
	Categories []*Category
	History    *history.History
	LogCloser  io.Closer // non-nil when logging to a file; closed on exit
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

// CategoryFilter holds the optional filtering rules for a category
type CategoryFilter struct {
	Regex         string         `yaml:"regex" mapstructure:"regex"`
	Glob          string         `yaml:"glob" mapstructure:"glob"`
	Ignore        []string       `yaml:"ignore" mapstructure:"ignore"`
	MinAge        time.Duration  `yaml:"min-age" mapstructure:"min-age"`
	MaxAge        time.Duration  `yaml:"max-age" mapstructure:"max-age"`
	MinSize       string         `yaml:"min-size" mapstructure:"min-size"`
	MaxSize       string         `yaml:"max-size" mapstructure:"max-size"`
	CompiledRegex *regexp.Regexp `yaml:"-" mapstructure:"-"` // compiled from Regex
	MinSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MinSize
	MaxSizeBytes  int64          `yaml:"-" mapstructure:"-"` // parsed from MaxSize
}

// Category represents a file category with its properties
type Category struct {
	Name             string         `yaml:"name" mapstructure:"name"`
	Extensions       []string       `yaml:"extensions" mapstructure:"extensions"`
	Source           string         `yaml:"source" mapstructure:"source"`
	Destination      string         `yaml:"destination" mapstructure:"destination"`
	ConflictStrategy string         `yaml:"conflict-strategy" mapstructure:"conflict-strategy"`
	Filter           CategoryFilter `yaml:"filter" mapstructure:"filter"`
}

// ConfigOption is a function that modifies the configuration
type ConfigOption func(*Config)

// WithOutput prompts the user to specify the output
func WithOutput() ConfigOption {
	terminal.ClearScreen()
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
	terminal.ClearScreen()
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
	terminal.ClearScreen()
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
	terminal.ClearScreen()
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
	terminal.ClearScreen()

	var want bool
	err := huh.NewConfirm().
		Title("Do you want to add categories?").
		Value(&want).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	categories := collectCategoryEntries(want)

	return func(c *Config) {
		c.Categories = append(c.Categories, categories...)
	}
}

// collectCategoryEntries collects one or more categories interactively when want is true.
func collectCategoryEntries(want bool) []Category {
	if !want {
		return nil
	}
	var categories []Category
	for {
		terminal.ClearScreen()
		cat := promptCategoryEntry()
		categories = append(categories, cat)

		var addMore bool
		if err := huh.NewConfirm().Title("Do you want to add another category?").Value(&addMore).Run(); err == huh.ErrUserAborted {
			os.Exit(0)
		}
		if !addMore {
			break
		}
	}
	return categories
}

// promptCategoryEntry collects a single category's fields from the user.
func promptCategoryEntry() Category {
	var name, source, destination, regex string

	if err := huh.NewInput().Title("Specify the category name").Value(&name).Run(); err == huh.ErrUserAborted {
		os.Exit(0)
	}
	if err := huh.NewInput().Title("Specify the source directory").Value(&source).Run(); err == huh.ErrUserAborted {
		os.Exit(0)
	}
	if err := huh.NewInput().Title("Specify the destination directory").Value(&destination).Run(); err == huh.ErrUserAborted {
		os.Exit(0)
	}

	extensions := promptExtensionsOrRegex(&regex)

	return Category{
		Name:        name,
		Extensions:  extensions,
		Source:      source,
		Destination: destination,
		Filter: CategoryFilter{
			Regex: regex,
		},
	}
}

// promptExtensionsOrRegex asks whether to use regex or extensions and collects the chosen input.
func promptExtensionsOrRegex(regex *string) []string {
	var useRegex bool
	if err := huh.NewConfirm().Title("Do you want to use Regex for filtering?").Value(&useRegex).Run(); err == huh.ErrUserAborted {
		os.Exit(0)
	}
	if useRegex {
		if err := huh.NewInput().Title("Specify the Regex pattern").Value(regex).Run(); err == huh.ErrUserAborted {
			os.Exit(0)
		}
		return nil
	}
	return promptExtensions()
}

// promptExtensions asks the user to optionally add file extensions one by one.
func promptExtensions() []string {
	var wantExtensions bool
	if err := huh.NewConfirm().Title("Do you want to add extensions?").Value(&wantExtensions).Run(); err == huh.ErrUserAborted {
		os.Exit(0)
	}
	if !wantExtensions {
		return nil
	}
	var extensions []string
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
	return extensions
}
