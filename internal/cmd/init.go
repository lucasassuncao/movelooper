// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	initForce       bool
	initInteractive bool
	initTemplate    string
)

// InitCmd generates a configuration file
func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize movelooper configuration",
		Long: `Initialize movelooper configuration file with predefined templates or interactive mode.
		
Available templates:
  - basic:    Simple configuration with one category (images)
  - media:    Configuration for organizing media files (images, videos, audio)
  - dev:      Configuration for organizing development files (code, docs, configs)
  - full:     Complete example with multiple categories
  
The configuration file will be created at: <executable_dir>/conf/movelooper.yaml`,
		Example: `  # Interactive mode (recommended for first time)
  movelooper init -i
  
  # Use a template
  movelooper init -t media
  
  # Force overwrite existing config
  movelooper init -f`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error getting executable: %v", err)
		}

		configPath := filepath.Join(filepath.Dir(ex), "conf")
		configFile := filepath.Join(configPath, "movelooper.yaml")

		if _, err := os.Stat(configFile); err == nil && !initForce {
			pterm.Error.Printf("Configuration file already exists at: %s\n", configFile)
			pterm.Info.Println("Use --force to overwrite")
			return nil
		}

		if err := helper.CreateDirectory(configPath); err != nil {
			return fmt.Errorf("error creating config directory: %v", err)
		}

		var config *models.Config

		if initInteractive {
			config = generateInteractiveConfig()
		} else {
			config = getTemplateConfig(initTemplate)
		}

		// Write config to file
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("error marshaling config: %v", err)
		}

		if err := os.WriteFile(configFile, data, 0644); err != nil {
			return fmt.Errorf("error writing config file: %v", err)
		}

		clearScreen()
		pterm.Success.Printf("Configuration file created at: %s\n", configFile)
		pterm.Info.Println("\nNext steps:")
		pterm.Info.Println("  1. Edit the configuration file to customize categories")
		pterm.Info.Println("  2. Run 'movelooper' to organize your files")
		pterm.Info.Println("  3. Run 'movelooper --dry-run' to see what would be moved")
		pterm.Info.Println("  4. Run 'movelooper watch' to continuously watch for file changes and move them")
		pterm.Info.Println("  5. Run 'movelooper --help' to see all available commands")

		return nil
	}

	cmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing configuration file")
	cmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Interactive mode with prompts")
	cmd.Flags().StringVarP(&initTemplate, "template", "t", "basic", "Template to use (basic, media, dev, full)")

	return cmd
}

// generateInteractiveConfig creates configuration through interactive prompts
func generateInteractiveConfig() *models.Config {
	clearScreen()
	config := &models.Config{
		Configuration: models.Configuration{},
		Categories:    []models.Category{},
	}

	pterm.DefaultSection.Println("Logging Configuration")

	var output string
	err := huh.NewSelect[string]().
		Title("Where should logs be output?").
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
	config.Configuration.Output = output

	if output == "log" || output == "file" || output == "both" {
		defaultLogPath := getDefaultLogPath()
		var logFile string
		err := huh.NewInput().
			Title("Log file path").
			Value(&logFile).
			Placeholder(defaultLogPath).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if logFile == "" {
			logFile = defaultLogPath
		}
		config.Configuration.LogFile = logFile
	}

	var logLevel string
	err = huh.NewSelect[string]().
		Title("Log level").
		Options(
			huh.NewOption("Trace", "trace"),
			huh.NewOption("Debug", "debug"),
			huh.NewOption("Info", "info"),
			huh.NewOption("Warn", "warn"),
			huh.NewOption("Error", "error"),
			huh.NewOption("Fatal", "fatal"),
		).
		Value(&logLevel).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}
	config.Configuration.LogLevel = logLevel

	var showCaller bool
	err = huh.NewConfirm().
		Title("Show caller information in logs?").
		Value(&showCaller).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}
	config.Configuration.ShowCaller = showCaller

	var watchDelayStr string
	err = huh.NewInput().
		Title("Watch delay (e.g., 5m, 30s) - This is the delay between file changes before moving files").
		Value(&watchDelayStr).
		Placeholder("5m").
		Validate(func(str string) error {
			if str == "" {
				return nil
			}
			_, err := time.ParseDuration(str)
			return err
		}).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	if watchDelayStr == "" {
		watchDelayStr = "5m"
	}
	watchDelay, _ := time.ParseDuration(watchDelayStr)
	config.Configuration.WatchDelay = watchDelay

	clearScreen()
	pterm.DefaultSection.Println("Categories Configuration")
	pterm.Info.Println("Categories define how files are organized")
	pterm.Println()

	var addCategories bool
	err = huh.NewConfirm().
		Title("Do you want to add categories now?").
		Value(&addCategories).
		Run()
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	if addCategories {
		config.Categories = collectCategories()
	}

	// Add default category if none were added
	if len(config.Categories) == 0 {
		config.Categories = append(config.Categories, getDefaultCategory())
	}

	return config
}

// collectCategories collects categories from user input
func collectCategories() []models.Category {
	var categories []models.Category

	for {
		clearScreen()
		var name string
		err := huh.NewInput().
			Title("Category name (e.g., images, documents)").
			Value(&name).
			Validate(func(str string) error {
				if str == "" {
					return fmt.Errorf("category name is required")
				}
				return nil
			}).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		var strategy string
		err = huh.NewSelect[string]().
			Title("Conflict strategy (if file exists)").
			Options(
				huh.NewOption("Rename", "rename"),
				huh.NewOption("Hash Check", "hash_check"),
				huh.NewOption("Overwrite", "overwrite"),
				huh.NewOption("Skip", "skip"),
			).
			Value(&strategy).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		var source string
		err = huh.NewInput().
			Title("Source directory").
			Value(&source).
			Placeholder(getDefaultSourcePath()).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if source == "" {
			source = getDefaultSourcePath()
		}

		var destination string
		err = huh.NewInput().
			Title("Destination directory").
			Value(&destination).
			Placeholder(getDefaultDestinationPath(name)).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if destination == "" {
			destination = getDefaultDestinationPath(name)
		}

		var useRegex, useSubfolder bool
		var regex string
		var extensions []string

		err = huh.NewConfirm().
			Title("Do you want to use Regex for filtering?").
			Value(&useRegex).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if useRegex {
			err = huh.NewInput().
				Title("Specify the Regex pattern").
				Value(&regex).
				Run()
			if err == huh.ErrUserAborted {
				os.Exit(0)
			}
		} else {
			extensions = collectExtensions(name)
		}

		err = huh.NewConfirm().
			Title("Create subfolders for extensions?").
			Description("If yes, files will be moved to Destination/Extension/File.ext").
			Value(&useSubfolder).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		category := models.Category{
			Name:                  name,
			Extensions:            extensions,
			Regex:                 regex,
			Source:                source,
			Destination:           destination,
			ConflictStrategy:      strategy,
			UseExtensionSubfolder: useSubfolder,
		}

		categories = append(categories, category)

		// Summary
		pterm.Println()
		pterm.DefaultSection.Println("Category Summary")
		printCategorySummary(category)
		pterm.Println()

		// Add more?
		var addMore bool
		err = huh.NewConfirm().
			Title("Add another category?").
			Value(&addMore).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if !addMore {
			break
		}
	}
	return categories
}

// collectExtensions collects file extensions from user input
func collectExtensions(categoryName string) []string {
	var extensions []string

	suggestions := getExtensionSuggestions(categoryName)
	if len(suggestions) > 0 {
		pterm.Info.Printf("Suggested extensions for '%s': %s\n", categoryName, strings.Join(suggestions, ", "))
		var useSuggestions bool
		err := huh.NewConfirm().
			Title("Use suggested extensions?").
			Value(&useSuggestions).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if useSuggestions {
			return suggestions
		}
	}

	pterm.Info.Println("Enter file extensions (without the dot, e.g., 'jpg' not '.jpg')")

	for {
		var extension string
		err := huh.NewInput().
			Title("File extension").
			Value(&extension).
			Run()
		if err == huh.ErrUserAborted {
			os.Exit(0)
		}

		if extension != "" {
			// Remove dot if user added it
			extension = strings.TrimPrefix(extension, ".")
			extensions = append(extensions, extension)
		}

		if len(extensions) > 0 {
			var addMore bool
			err := huh.NewConfirm().
				Title("Add another extension?").
				Value(&addMore).
				Run()
			if err == huh.ErrUserAborted {
				os.Exit(0)
			}

			if !addMore {
				break
			}
		}
	}
	return extensions
}

// getDefaultCategory returns a default category configuration
func getDefaultCategory() models.Category {
	source := getDefaultSourcePath()
	return models.Category{
		Name:             "images",
		Extensions:       []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		Source:           source,
		Destination:      filepath.Join(source, "images"),
		ConflictStrategy: "rename",
	}
}

// getTemplateConfig returns a predefined template configuration
func getTemplateConfig(template string) *models.Config {
	templates := map[string]func() *models.Config{
		"basic":      getBasicTemplate,
		"music":      getMusicTemplate,
		"video":      getVideoTemplate,
		"books":      getBooksTemplate,
		"images":     getImagesTemplate,
		"archives":   getArchivesTemplate,
		"installers": getInstallersTemplate,
		"regex":      getRegexTemplate,
		"full":       getFullTemplate,
	}

	templateFunc, exists := templates[template]
	if !exists {
		pterm.Warning.Printf("Unknown template '%s', using 'basic'\n", template)
		templateFunc = getBasicTemplate
	}

	return templateFunc()
}

// getBasicTemplate returns the basic configuration template
func getBasicTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "images",
				Extensions:            []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "images"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getMusicTemplate returns the music configuration template
func getMusicTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "music",
				Extensions:            []string{"mp3", "wav", "flac", "aac"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "music"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getVideoTemplate returns the video configuration template
func getVideoTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "videos",
				Extensions:            []string{"mp4", "avi", "mkv", "mov", "wmv"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "videos"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getImagesTemplate returns the images configuration template
func getImagesTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "images",
				Extensions:            []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "images"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getBooksTemplate returns the books configuration template
func getBooksTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "books",
				Extensions:            []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "books"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getArchivesTemplate returns the archives configuration template
func getArchivesTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "archives",
				Extensions:            []string{"zip", "tar", "gz", "bz2", "rar", "7z"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "archives"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getInstallersTemplate returns the installers configuration template
func getInstallersTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "installers",
				Extensions:            []string{"exe", "msi", "apk"},
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "installers"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getRegexTemplate returns the regex configuration template
func getRegexTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "regex",
				Regex:                 ".*",
				Source:                getDefaultSourcePath(),
				Destination:           filepath.Join(getDefaultSourcePath(), "regex"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getFullTemplate returns the full configuration template
func getFullTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "info",
			ShowCaller: true,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:                  "images",
				Extensions:            []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "images"),
				ConflictStrategy:      "rename",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "videos",
				Extensions:            []string{"mp4", "avi", "mkv", "mov", "wmv"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "videos"),
				ConflictStrategy:      "overwrite",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "music",
				Extensions:            []string{"mp3", "wav", "flac", "aac"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "music"),
				ConflictStrategy:      "skip",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "books",
				Extensions:            []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "books"),
				ConflictStrategy:      "hash_check",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "archives",
				Extensions:            []string{"zip", "tar", "gz", "bz2", "rar", "7z"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "archives"),
				ConflictStrategy:      "hash_check",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "installers",
				Extensions:            []string{"exe", "msi", "apk"},
				Source:                basePath,
				Destination:           filepath.Join(basePath, "installers"),
				ConflictStrategy:      "hash_check",
				UseExtensionSubfolder: true,
			},
			{
				Name:                  "regex",
				Regex:                 ".*",
				Source:                basePath,
				Destination:           filepath.Join(basePath, "regex"),
				ConflictStrategy:      "hash_check",
				UseExtensionSubfolder: true,
			},
		},
	}
}

// getDefaultSourcePath returns the default source path (Downloads folder)
func getDefaultSourcePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\Downloads"
	}
	return filepath.Join(homeDir, "Downloads")
}

// getDefaultDestinationPath returns the default destination path for a category
func getDefaultDestinationPath(categoryName string) string {
	source := getDefaultSourcePath()
	return filepath.Join(source, categoryName)
}

// getDefaultLogPath returns the default log file path
func getDefaultLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\logs\\movelooper.log"
	}
	return filepath.Join(homeDir, ".movelooper", "logs", "movelooper.log")
}

// getExtensionSuggestions provides extension suggestions based on category name
func getExtensionSuggestions(categoryName string) []string {
	suggestions := map[string][]string{
		"images":     {"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		"photos":     {"jpg", "jpeg", "png", "raw", "cr2", "nef"},
		"videos":     {"mp4", "avi", "mkv", "mov", "wmv"},
		"music":      {"mp3", "wav", "flac", "aac", "ogg"},
		"documents":  {"pdf", "doc", "docx", "txt", "md"},
		"archives":   {"zip", "tar", "gz", "rar", "7z"},
		"installers": {"exe", "msi", "apk"},
	}

	name := strings.ToLower(categoryName)
	for key, exts := range suggestions {
		if strings.Contains(name, key) {
			return exts
		}
	}

	return nil
}

// printCategorySummary prints a summary of the category configuration
func printCategorySummary(category models.Category) {
	pterm.Printf("  Name:        %s\n", pterm.Cyan(category.Name))
	pterm.Printf("  Strategy:    %s\n", pterm.Magenta(category.ConflictStrategy))
	pterm.Printf("  Source:      %s\n", pterm.Yellow(category.Source))
	pterm.Printf("  Destination: %s\n", pterm.Yellow(category.Destination))
	pterm.Printf("  Extensions:  %s\n", pterm.Green(strings.Join(category.Extensions, ", ")))
}

// clearScreen clears the terminal screen
func clearScreen() {
	pterm.Println()
}
