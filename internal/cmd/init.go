// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/terminal"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// initOptions holds the flag values for the init command.
type initOptions struct {
	force       bool
	interactive bool
	template    string
	output      string
}

// InitCmd generates a configuration file
func InitCmd() *cobra.Command {
	opts := initOptions{}

	cmd := &cobra.Command{
		Use:               "init",
		Short:             "Initialize movelooper configuration",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Initialize movelooper configuration file with predefined templates or interactive mode.

Available templates:
  - basic:       Simple configuration with one category (images)
  - images:      Configuration for organizing image files
  - music:       Configuration for organizing music files
  - video:       Configuration for organizing video files
  - books:       Configuration for organizing book/document files
  - archives:    Configuration for organizing archive files
  - installers:  Configuration for organizing installer files
  - regex:       Example using regex name filtering
  - full:        Complete example with multiple categories and all options

By default the configuration file is created at: <executable_dir>/conf/movelooper.yaml`,
		Example: `  # Interactive mode (recommended for first time)
  movelooper init -i

  # Use a template
  movelooper init -t media

  # Save to a custom path
  movelooper init -o /path/to/movelooper.yaml

  # Force overwrite existing config
  movelooper init -f`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing configuration file")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode with prompts")
	cmd.Flags().StringVarP(&opts.template, "template", "t", "basic", "Template to use (basic, images, music, video, books, archives, installers, regex, full)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Path to write the configuration file (default: <executable_dir>/conf/movelooper.yaml)")

	return cmd
}

// runInit executes the init command with the given options.
func runInit(opts initOptions) error {
	var configFile string
	if opts.output != "" {
		configFile = opts.output
	} else {
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error getting executable: %v", err)
		}
		configFile = filepath.Join(filepath.Dir(ex), "conf", "movelooper.yaml")
	}

	configPath := filepath.Dir(configFile)

	if _, err := os.Stat(configFile); err == nil && !opts.force {
		pterm.Error.Printf("Configuration file already exists at: %s\n", configFile)
		pterm.Info.Println("Use --force to overwrite")
		return nil
	}

	if err := helper.CreateDirectory(configPath); err != nil {
		return fmt.Errorf("error creating config directory: %v", err)
	}

	var config *models.Config
	if opts.interactive {
		config = generateInteractiveConfig()
	} else {
		config = getTemplateConfig(opts.template)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	terminal.ClearScreen()
	fmt.Printf("Config created: %s\n", configFile)

	return nil
}

// exitIfAborted exits cleanly when the user cancels an interactive prompt.
// huh returns ErrUserAborted on Ctrl+C / Esc; we treat that as a graceful exit.
func exitIfAborted(err error) {
	if err == huh.ErrUserAborted {
		os.Exit(0)
	}
}

// generateInteractiveConfig creates configuration through interactive prompts
func generateInteractiveConfig() *models.Config {
	terminal.ClearScreen()
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
	exitIfAborted(err)
	config.Configuration.Output = output

	if output == "log" || output == "file" || output == "both" {
		defaultLogPath := getDefaultLogPath()
		var logFile string
		err := huh.NewInput().
			Title("Log file path").
			Value(&logFile).
			Placeholder(defaultLogPath).
			Run()
		exitIfAborted(err)

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
	exitIfAborted(err)
	config.Configuration.LogLevel = logLevel

	var showCaller bool
	err = huh.NewConfirm().
		Title("Show caller information in logs?").
		Value(&showCaller).
		Run()
	exitIfAborted(err)
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
	exitIfAborted(err)

	if watchDelayStr == "" {
		watchDelayStr = "5m"
	}
	watchDelay, _ := time.ParseDuration(watchDelayStr)
	config.Configuration.WatchDelay = watchDelay

	terminal.ClearScreen()
	pterm.DefaultSection.Println("Categories Configuration")
	pterm.Info.Println("Categories define how files are organized")
	pterm.Println()

	var addCategories bool
	err = huh.NewConfirm().
		Title("Do you want to add categories now?").
		Value(&addCategories).
		Run()
	exitIfAborted(err)

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
		category := promptOneCategory()
		categories = append(categories, category)

		pterm.Println()
		pterm.DefaultSection.Println("Category Summary")
		printCategorySummary(category)
		pterm.Println()

		var addMore bool
		exitIfAborted(huh.NewConfirm().Title("Add another category?").Value(&addMore).Run())
		if !addMore {
			break
		}
	}
	return categories
}

// promptOneCategory prompts the user for all fields of a single category.
func promptOneCategory() models.Category {
	terminal.ClearScreen()

	var name string
	exitIfAborted(huh.NewInput().
		Title("Category name (e.g., images, documents)").
		Value(&name).
		Validate(func(str string) error {
			if str == "" {
				return fmt.Errorf("category name is required")
			}
			return nil
		}).Run())

	var strategy string
	exitIfAborted(huh.NewSelect[string]().
		Title("Conflict strategy (if file exists)").
		Options(
			huh.NewOption("Rename", "rename"),
			huh.NewOption("Hash Check", "hash_check"),
			huh.NewOption("Overwrite", "overwrite"),
			huh.NewOption("Skip", "skip"),
		).Value(&strategy).Run())

	source := promptWithDefault(
		huh.NewInput().Title("Source directory").Placeholder(getDefaultSourcePath()),
		getDefaultSourcePath(),
	)
	destination := promptWithDefault(
		huh.NewInput().Title("Destination directory").Placeholder(getDefaultDestinationPath(name)),
		getDefaultDestinationPath(name),
	)

	extensions := collectExtensions(name)
	regex, glob := promptNameFilter()
	ignorePatterns := promptIgnorePatterns()
	minAge, maxAge := promptAgeFilter()
	minSize, maxSize := promptSizeFilter()

	var organizeBy string
	exitIfAborted(huh.NewInput().
		Title("Organize-by template (optional)").
		Description(`Organize files into subdirectories using a template.
Examples: {ext}  |  {ext}/{mod-year}  |  {mod-year}/{mod-month}/{mod-day}
Leave blank to move files directly into destination.`).
		Value(&organizeBy).
		Run())

	return models.Category{
		Name: name,
		Source: models.CategorySource{
			Path:       source,
			Extensions: extensions,
			Filter: models.CategoryFilter{
				Regex:   regex,
				Glob:    glob,
				Ignore:  ignorePatterns,
				MinAge:  minAge,
				MaxAge:  maxAge,
				MinSize: minSize,
				MaxSize: maxSize,
			},
		},
		Destination: models.CategoryDestination{
			Path:             destination,
			OrganizeBy:       organizeBy,
			ConflictStrategy: strategy,
		},
	}
}

// promptWithDefault runs a huh.Input field and returns the default value when the user leaves it blank.
func promptWithDefault(field *huh.Input, defaultVal string) string {
	var val string
	field.Value(&val)
	exitIfAborted(field.Run())
	if val == "" {
		return defaultVal
	}
	return val
}

// promptNameFilter asks the user to choose an optional name filter (regex or glob).
func promptNameFilter() (regex, glob string) {
	var filterType string
	exitIfAborted(huh.NewSelect[string]().
		Title("Add an optional name filter?").
		Description("Extensions already define the file type; this further filters by name").
		Options(
			huh.NewOption("None", "none"),
			huh.NewOption("Glob pattern (e.g., report_*.pdf, invoice_*.{pdf,docx})", "glob"),
			huh.NewOption("Regex pattern (e.g., ^report_\\d{4}\\.pdf$)", "regex"),
		).Value(&filterType).Run())

	switch filterType {
	case "regex":
		exitIfAborted(huh.NewInput().
			Title("Specify the Regex pattern").
			Value(&regex).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("regex pattern is required")
				}
				_, err := regexp.Compile(s)
				return err
			}).Run())
	case "glob":
		exitIfAborted(huh.NewInput().
			Title("Specify the Glob pattern").
			Description("Use * for any characters, ? for one character, {a,b} for alternatives").
			Value(&glob).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("glob pattern is required")
				}
				return helper.ValidateGlob(s)
			}).Run())
	}
	return regex, glob
}

// promptIgnorePatterns asks the user whether to add ignore patterns and collects them.
func promptIgnorePatterns() []string {
	var addIgnore bool
	exitIfAborted(huh.NewConfirm().
		Title("Do you want to add ignore patterns?").
		Description("Glob patterns for files to skip (e.g., *_temp.*, screenshot_*)").
		Value(&addIgnore).Run())
	if addIgnore {
		return collectIgnorePatterns()
	}
	return nil
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
		exitIfAborted(err)

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
		exitIfAborted(err)

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
			exitIfAborted(err)

			if !addMore {
				break
			}
		}
	}
	return extensions
}

// promptAgeFilter asks the user for optional min-age and max-age filters.
func promptAgeFilter() (minAge, maxAge time.Duration) {
	validateDuration := func(s string) error {
		if s == "" {
			return nil
		}
		_, err := time.ParseDuration(s)
		return err
	}

	var minAgeStr string
	exitIfAborted(huh.NewInput().
		Title("Minimum file age before moving (e.g., 24h, 168h) — leave blank to disable").
		Description("Only files older than this duration will be moved").
		Value(&minAgeStr).
		Validate(validateDuration).
		Run())

	var maxAgeStr string
	exitIfAborted(huh.NewInput().
		Title("Maximum file age before moving (e.g., 720h, 8760h) — leave blank to disable").
		Description("Only files newer than this duration will be moved").
		Value(&maxAgeStr).
		Validate(validateDuration).
		Run())

	if minAgeStr != "" {
		minAge, _ = time.ParseDuration(minAgeStr)
	}
	if maxAgeStr != "" {
		maxAge, _ = time.ParseDuration(maxAgeStr)
	}
	return minAge, maxAge
}

// promptSizeFilter asks the user for optional min-size and max-size filters.
func promptSizeFilter() (minSize, maxSize string) {
	exitIfAborted(huh.NewInput().
		Title("Minimum file size before moving (e.g., 1MB, 500KB) — leave blank to disable").
		Description("Only files larger than this size will be moved").
		Value(&minSize).Run())

	exitIfAborted(huh.NewInput().
		Title("Maximum file size before moving (e.g., 10MB, 1GB) — leave blank to disable").
		Description("Only files smaller than this size will be moved").
		Value(&maxSize).Run())

	return minSize, maxSize
}

// collectIgnorePatterns collects glob ignore patterns from user input
func collectIgnorePatterns() []string {
	var patterns []string

	pterm.Info.Println("Enter glob patterns for files to ignore (e.g., *_temp.*, screenshot_*, *.tmp)")

	for {
		var pattern string
		err := huh.NewInput().
			Title("Ignore pattern").
			Value(&pattern).
			Run()
		exitIfAborted(err)

		if pattern != "" {
			patterns = append(patterns, pattern)
		}

		if len(patterns) > 0 {
			var addMore bool
			err := huh.NewConfirm().
				Title("Add another ignore pattern?").
				Value(&addMore).
				Run()
			exitIfAborted(err)

			if !addMore {
				break
			}
		}
	}
	return patterns
}

// getDefaultCategory returns a default category configuration
func getDefaultCategory() models.Category {
	source := getDefaultSourcePath()
	return models.Category{
		Name: "images",
		Source: models.CategorySource{
			Path:       source,
			Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
		},
		Destination: models.CategoryDestination{
			Path:             filepath.Join(source, "images"),
			ConflictStrategy: "rename",
		},
	}
}

// simpleTemplateDef describes a single-category template whose configuration
// is always the same (console output, info level, rename strategy).
// Adding a new simple template only requires a new entry in simpleTemplateDefs.
type simpleTemplateDef struct {
	categoryName string
	extensions   []string
}

// simpleTemplateDefs holds the data for all single-category templates.
var simpleTemplateDefs = map[string]simpleTemplateDef{
	"basic":      {categoryName: "images", extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"}},
	"images":     {categoryName: "images", extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"}},
	"music":      {categoryName: "music", extensions: []string{"mp3", "wav", "flac", "aac"}},
	"video":      {categoryName: "videos", extensions: []string{"mp4", "avi", "mkv", "mov", "wmv"}},
	"books":      {categoryName: "books", extensions: []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"}},
	"archives":   {categoryName: "archives", extensions: []string{"zip", "tar", "gz", "bz2", "rar", "7z"}},
	"installers": {categoryName: "installers", extensions: []string{"exe", "msi", "apk"}},
}

// buildSimpleTemplate constructs a Config from a simpleTemplateDef.
func buildSimpleTemplate(def simpleTemplateDef) *models.Config {
	src := getDefaultSourcePath()
	return &models.Config{
		Configuration: models.Configuration{
			Output:     "console",
			LogLevel:   "info",
			ShowCaller: false,
			WatchDelay: 5 * time.Minute,
		},
		Categories: []models.Category{
			{
				Name: def.categoryName,
				Source: models.CategorySource{
					Path:       src,
					Extensions: def.extensions,
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(src, def.categoryName),
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
			},
		},
	}
}

// getTemplateConfig returns a predefined template configuration.
func getTemplateConfig(template string) *models.Config {
	if def, ok := simpleTemplateDefs[template]; ok {
		return buildSimpleTemplate(def)
	}
	switch template {
	case "regex":
		return getRegexTemplate()
	case "full":
		return getFullTemplate()
	default:
		pterm.Warning.Printf("Unknown template '%s', using 'basic'\n", template)
		return buildSimpleTemplate(simpleTemplateDefs["basic"])
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
				Name: "regex",
				Source: models.CategorySource{
					Path:       getDefaultSourcePath(),
					Extensions: []string{"pdf", "txt", "log"},
					Filter: models.CategoryFilter{
						Regex: `^\d{4}-\d{2}-\d{2}_.*`,
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(getDefaultSourcePath(), "regex"),
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
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
				Name: "images",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"},
					Filter: models.CategoryFilter{
						Ignore: []string{"screenshot_*", "*_temp.*"},
						MinAge: 24 * time.Hour,
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "images"),
					ConflictStrategy: "rename",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "videos",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"mp4", "avi", "mkv", "mov", "wmv"},
					Filter: models.CategoryFilter{
						Ignore:  []string{"*_preview.*", "*_draft.*"},
						MinSize: "100MB",
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "videos"),
					ConflictStrategy: "overwrite",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "music",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"mp3", "wav", "flac", "aac"},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "music"),
					ConflictStrategy: "skip",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "books",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"pdf", "epub", "mobi", "azw3", "doc", "docx"},
					Filter: models.CategoryFilter{
						MinSize: "1MB",
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "books"),
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "archives",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"zip", "tar", "gz", "bz2", "rar", "7z"},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "archives"),
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "installers",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"exe", "msi", "apk"},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "installers"),
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "dated-docs",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"pdf", "txt", "log"},
					Filter: models.CategoryFilter{
						Regex: `^\d{4}-\d{2}-\d{2}_.*`,
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "dated"),
					ConflictStrategy: "hash_check",
					OrganizeBy:       "{ext}",
				},
			},
			{
				Name: "reports",
				Source: models.CategorySource{
					Path:       basePath,
					Extensions: []string{"pdf", "docx"},
					Filter: models.CategoryFilter{
						Glob: "report_*",
					},
				},
				Destination: models.CategoryDestination{
					Path:             filepath.Join(basePath, "reports"),
					ConflictStrategy: "rename",
				},
			},
		},
	}
}

// getDefaultSourcePath returns the default source path (Downloads folder).
// Falls back to the OS temp directory when the home directory is unavailable.
func getDefaultSourcePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "movelooper", "downloads")
	}
	return filepath.Join(homeDir, "Downloads")
}

// getDefaultDestinationPath returns the default destination path for a category
func getDefaultDestinationPath(categoryName string) string {
	source := getDefaultSourcePath()
	return filepath.Join(source, categoryName)
}

// getDefaultLogPath returns the default log file path.
// Falls back to the OS temp directory when the home directory is unavailable.
func getDefaultLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "movelooper", "logs", "movelooper.log")
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
	pterm.Printf("  Name:              %s\n", pterm.Cyan(category.Name))
	pterm.Printf("  Enabled:           %s\n", pterm.Yellow(fmt.Sprintf("%v", category.IsEnabled())))
	pterm.Printf("  Source:            %s\n", pterm.Yellow(category.Source.Path))
	pterm.Printf("  Extensions:        %s\n", pterm.Green(strings.Join(category.Source.Extensions, ", ")))
	pterm.Printf("  Destination:       %s\n", pterm.Yellow(category.Destination.Path))
	pterm.Printf("  Strategy:          %s\n", pterm.Magenta(category.Destination.ConflictStrategy))
	organizeByDisplay := category.Destination.OrganizeBy
	if organizeByDisplay == "" {
		organizeByDisplay = "(none)"
	}
	pterm.Printf("  Organize by:       %s\n", pterm.Yellow(organizeByDisplay))
	f := category.Source.Filter
	if f.Regex != "" {
		pterm.Printf("  Regex:             %s\n", pterm.Green(f.Regex))
	}
	if f.Glob != "" {
		pterm.Printf("  Glob:              %s\n", pterm.Green(f.Glob))
	}
	if len(f.Ignore) > 0 {
		pterm.Printf("  Ignore:            %s\n", pterm.Red(strings.Join(f.Ignore, ", ")))
	}
	if f.MinAge > 0 {
		pterm.Printf("  Min Age:           %s\n", pterm.Yellow(f.MinAge.String()))
	}
	if f.MaxAge > 0 {
		pterm.Printf("  Max Age:           %s\n", pterm.Yellow(f.MaxAge.String()))
	}
	if f.MinSize != "" {
		pterm.Printf("  Min Size:          %s\n", pterm.Yellow(f.MinSize))
	}
	if f.MaxSize != "" {
		pterm.Printf("  Max Size:          %s\n", pterm.Yellow(f.MaxSize))
	}
}
