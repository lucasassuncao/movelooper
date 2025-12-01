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

		extensions := collectExtensions(name)

		category := models.Category{
			Name:             name,
			Extensions:       extensions,
			Source:           source,
			Destination:      destination,
			ConflictStrategy: strategy, // Preenchendo o novo campo
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
		"basic": getBasicTemplate,
		"media": getMediaTemplate,
		"dev":   getDevTemplate,
		"full":  getFullTemplate,
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
				Name:             "images",
				Extensions:       []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
				Source:           getDefaultSourcePath(),
				Destination:      filepath.Join(getDefaultSourcePath(), "images"),
				ConflictStrategy: "rename",
			},
		},
	}
}

// getMediaTemplate returns the media configuration template
func getMediaTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:             "images",
				Extensions:       []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "images"),
				ConflictStrategy: "hash_check",
			},
			{
				Name:             "videos",
				Extensions:       []string{"mp4", "avi", "mkv", "mov"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "videos"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "audio",
				Extensions:       []string{"mp3", "wav", "flac"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "audio"),
				ConflictStrategy: "rename",
			},
		},
	}
}

// getMediaTemplate returns the media configuration template
func getDevTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "debug",
			ShowCaller: true,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name:             "source-code",
				Extensions:       []string{"go", "py", "js", "ts", "java", "cpp", "c", "rs", "rb"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "code"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "documentation",
				Extensions:       []string{"md", "txt", "pdf", "doc", "docx"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "docs"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "configs",
				Extensions:       []string{"yaml", "yml", "json", "toml", "xml", "ini", "conf"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "configs"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "archives",
				Extensions:       []string{"zip", "tar", "gz", "rar", "7z"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "archives"),
				ConflictStrategy: "hash_check",
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
				Name:             "images",
				Extensions:       []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "images"),
				ConflictStrategy: "hash_check",
			},
			{
				Name:             "videos",
				Extensions:       []string{"mp4", "avi", "mkv", "mov", "wmv"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "videos"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "audio",
				Extensions:       []string{"mp3", "wav", "flac", "aac"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "audio"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "documents",
				Extensions:       []string{"pdf", "doc", "docx", "txt", "md"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "documents"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "spreadsheets",
				Extensions:       []string{"xls", "xlsx", "csv"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "spreadsheets"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "presentations",
				Extensions:       []string{"ppt", "pptx"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "presentations"),
				ConflictStrategy: "rename",
			},
			{
				Name:             "archives",
				Extensions:       []string{"zip", "tar", "gz", "rar", "7z"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "archives"),
				ConflictStrategy: "hash_check",
			},
			{
				Name:             "executables",
				Extensions:       []string{"exe", "msi", "dmg", "app"},
				Source:           basePath,
				Destination:      filepath.Join(basePath, "executables"),
				ConflictStrategy: "overwrite",
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
		"image":        {"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		"images":       {"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		"photo":        {"jpg", "jpeg", "png", "raw", "cr2", "nef"},
		"photos":       {"jpg", "jpeg", "png", "raw", "cr2", "nef"},
		"video":        {"mp4", "avi", "mkv", "mov", "wmv"},
		"videos":       {"mp4", "avi", "mkv", "mov", "wmv"},
		"audio":        {"mp3", "wav", "flac", "aac", "ogg"},
		"music":        {"mp3", "wav", "flac", "aac", "ogg"},
		"document":     {"pdf", "doc", "docx", "txt", "md"},
		"documents":    {"pdf", "doc", "docx", "txt", "md"},
		"code":         {"go", "py", "js", "ts", "java", "cpp"},
		"source":       {"go", "py", "js", "ts", "java", "cpp"},
		"archive":      {"zip", "tar", "gz", "rar", "7z"},
		"archives":     {"zip", "tar", "gz", "rar", "7z"},
		"spreadsheet":  {"xls", "xlsx", "csv"},
		"spreadsheets": {"xls", "xlsx", "csv"},
		"presentation": {"ppt", "pptx"},
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
