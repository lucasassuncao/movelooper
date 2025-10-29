// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		// Check if config already exists
		if _, err := os.Stat(configFile); err == nil && !initForce {
			pterm.Error.Printf("Configuration file already exists at: %s\n", configFile)
			pterm.Info.Println("Use --force to overwrite")
			return nil
		}

		// Create config directory
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
		pterm.Info.Println("  2. Run 'movelooper preview' to see what would be moved")
		pterm.Info.Println("  3. Run 'movelooper move' to organize your files")

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
	pterm.DefaultHeader.WithFullWidth().Println("Movelooper Configuration Generator")
	pterm.Println()

	config := &models.Config{
		Configuration: models.Configuration{},
		Categories:    []models.Category{},
	}

	// 1. Output configuration
	pterm.DefaultSection.Println("Logging Configuration")
	output, _ := pterm.DefaultInteractiveSelect.
		WithOptions([]string{"console", "log", "file", "both"}).
		WithDefaultText("Where should logs be output?").
		WithMaxHeight(10).
		Show()
	config.Configuration.Output = output

	// 2. Log file (if needed)
	if output == "log" || output == "file" || output == "both" {
		defaultLogPath := getDefaultLogPath()
		logFile, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Log file path").
			WithDefaultValue(defaultLogPath).
			Show()
		config.Configuration.LogFile = logFile
	}

	// 3. Log level
	logLevel, _ := pterm.DefaultInteractiveSelect.
		WithOptions([]string{"trace", "debug", "info", "warn", "error", "fatal"}).
		WithDefaultText("Log level").
		WithDefaultOption("info").
		WithMaxHeight(10).
		Show()
	config.Configuration.LogLevel = logLevel

	// 4. Show caller
	showCaller, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText("Show caller information in logs?").
		WithDefaultValue(false).
		Show()
	config.Configuration.ShowCaller = showCaller

	// 5. Categories
	clearScreen()
	pterm.DefaultSection.Println("Categories Configuration")
	pterm.Info.Println("Categories define how files are organized")
	pterm.Println()

	addCategories, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText("Do you want to add categories now?").
		WithDefaultValue(true).
		Show()

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
		pterm.DefaultHeader.WithFullWidth().Printf("Category #%d", len(categories)+1)
		pterm.Println()

		// Category name
		name, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Category name (e.g., images, documents, videos)").
			Show()

		if name == "" {
			pterm.Warning.Println("Category name is required")
			continue
		}

		// Source directory
		source, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Source directory (where files are located)").
			WithDefaultValue(getDefaultSourcePath()).
			Show()

		// Destination directory
		destination, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Destination directory (where files will be moved)").
			WithDefaultValue(getDefaultDestinationPath(name)).
			Show()

		// Extensions
		extensions := collectExtensions(name)

		category := models.Category{
			Name:        name,
			Extensions:  extensions,
			Source:      source,
			Destination: destination,
		}

		categories = append(categories, category)

		// Summary
		pterm.Println()
		pterm.DefaultSection.Println("Category Summary")
		printCategorySummary(category)
		pterm.Println()

		// Add more?
		addMore, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText("Add another category?").
			WithDefaultValue(false).
			Show()

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
		useSuggestions, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText("Use suggested extensions?").
			WithDefaultValue(true).
			Show()

		if useSuggestions {
			return suggestions
		}
	}

	pterm.Info.Println("Enter file extensions (without the dot, e.g., 'jpg' not '.jpg')")

	for {
		extension, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("File extension").
			Show()

		if extension != "" {
			// Remove dot if user added it
			extension = strings.TrimPrefix(extension, ".")
			extensions = append(extensions, extension)
		}

		if len(extensions) > 0 {
			addMore, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Add another extension?").
				WithDefaultValue(false).
				Show()

			if !addMore {
				break
			}
		}
	}

	return extensions
}

func getDefaultCategory() models.Category {
	source := getDefaultSourcePath()
	return models.Category{
		Name:        "images",
		Extensions:  []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		Source:      source,
		Destination: filepath.Join(source, "images"),
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

// Template functions

func getBasicTemplate() *models.Config {
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
		},
		Categories: []models.Category{
			{
				Name:        "images",
				Extensions:  []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
				Source:      getDefaultSourcePath(),
				Destination: filepath.Join(getDefaultSourcePath(), "images"),
			},
		},
	}
}

func getMediaTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "info",
			ShowCaller: false,
		},
		Categories: []models.Category{
			{
				Name:        "images",
				Extensions:  []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg", "tiff"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "images"),
			},
			{
				Name:        "videos",
				Extensions:  []string{"mp4", "avi", "mkv", "mov", "wmv", "flv", "webm", "m4v"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "videos"),
			},
			{
				Name:        "audio",
				Extensions:  []string{"mp3", "wav", "flac", "aac", "ogg", "wma", "m4a"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "audio"),
			},
		},
	}
}

func getDevTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "debug",
			ShowCaller: true,
		},
		Categories: []models.Category{
			{
				Name:        "source-code",
				Extensions:  []string{"go", "py", "js", "ts", "java", "cpp", "c", "rs", "rb"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "code"),
			},
			{
				Name:        "documentation",
				Extensions:  []string{"md", "txt", "pdf", "doc", "docx"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "docs"),
			},
			{
				Name:        "configs",
				Extensions:  []string{"yaml", "yml", "json", "toml", "xml", "ini", "conf"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "configs"),
			},
			{
				Name:        "archives",
				Extensions:  []string{"zip", "tar", "gz", "rar", "7z"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "archives"),
			},
		},
	}
}

func getFullTemplate() *models.Config {
	basePath := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "both",
			LogFile:    getDefaultLogPath(),
			LogLevel:   "info",
			ShowCaller: true,
		},
		Categories: []models.Category{
			{
				Name:        "images",
				Extensions:  []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "images"),
			},
			{
				Name:        "videos",
				Extensions:  []string{"mp4", "avi", "mkv", "mov", "wmv"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "videos"),
			},
			{
				Name:        "audio",
				Extensions:  []string{"mp3", "wav", "flac", "aac"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "audio"),
			},
			{
				Name:        "documents",
				Extensions:  []string{"pdf", "doc", "docx", "txt", "md"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "documents"),
			},
			{
				Name:        "spreadsheets",
				Extensions:  []string{"xls", "xlsx", "csv"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "spreadsheets"),
			},
			{
				Name:        "presentations",
				Extensions:  []string{"ppt", "pptx"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "presentations"),
			},
			{
				Name:        "archives",
				Extensions:  []string{"zip", "tar", "gz", "rar", "7z"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "archives"),
			},
			{
				Name:        "executables",
				Extensions:  []string{"exe", "msi", "dmg", "app"},
				Source:      basePath,
				Destination: filepath.Join(basePath, "executables"),
			},
		},
	}
}

// Helper functions

func getDefaultSourcePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\Downloads"
	}
	return filepath.Join(homeDir, "Downloads")
}

func getDefaultDestinationPath(categoryName string) string {
	source := getDefaultSourcePath()
	return filepath.Join(source, categoryName)
}

func getDefaultLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\logs\\movelooper.log"
	}
	return filepath.Join(homeDir, ".movelooper", "logs", "movelooper.log")
}

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

func printCategorySummary(category models.Category) {
	pterm.Printf("  Name:        %s\n", pterm.Cyan(category.Name))
	pterm.Printf("  Source:      %s\n", pterm.Yellow(category.Source))
	pterm.Printf("  Destination: %s\n", pterm.Yellow(category.Destination))
	pterm.Printf("  Extensions:  %s\n", pterm.Green(strings.Join(category.Extensions, ", ")))
}

func clearScreen() {
	// Reutiliza a função do models.Config se existir, ou implementa aqui
	pterm.Println()
}
