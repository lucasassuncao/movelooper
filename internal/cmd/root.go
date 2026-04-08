// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper, version string) *cobra.Command {
	var (
		dryRun    bool
		showFiles bool
	)

	cmd := &cobra.Command{
		Use:     "movelooper",
		Short:   "movelooper is a CLI tool for organizing and moving files",
		Version: version,
		Long: `movelooper organizes and moves files from source directories to destination directories,
based on configurable categories.

By default, it runs the move operation automatically.
Use --dry-run for a preview without moving files, and --show-files to display filenames.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return preRunHandler(m, configPath)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if m.LogCloser != nil {
				return m.LogCloser.Close()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(m, dryRun, showFiles)
		},
	}

	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (e.g., /path/to/movelooper.yaml)")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview mode! It shows what would be moved without moving files")
	cmd.PersistentFlags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")

	// Add subcommands
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(WatchCmd(m))
	cmd.AddCommand(UndoCmd(m))
	cmd.AddCommand(ConfigCmd(m))
	cmd.AddCommand(SelfUpdateCmd(version))

	return cmd
}

// movedSet tracks absolute paths that have already been moved in the current
// batch, preventing a file from being claimed by more than one category.
type movedSet map[string]bool

func (s movedSet) mark(dir, name string)     { s[filepath.Join(dir, name)] = true }
func (s movedSet) has(dir, name string) bool { return s[filepath.Join(dir, name)] }

// runStats accumulates totals across all categories for the end-of-run summary.
type runStats struct {
	totalFiles int
	totalBytes int64
	skipped    int
}

// runMove executes the default move operation across all configured categories.
func runMove(m *models.Movelooper, dryRun, showFiles bool) error {
	batchID := history.NewBatchID()
	moved := make(movedSet)
	var stats runStats

	for _, category := range m.Categories {
		if !category.IsEnabled() {
			m.Logger.Info(fmt.Sprintf("[%s] category disabled, skipping", category.Name))
			stats.skipped++
			continue
		}
		processCategoryMove(m, category, moved, batchID, dryRun, showFiles, &stats)
	}

	if dryRun {
		m.Logger.Info("dry-run complete, no files were moved",
			m.Logger.Args("matched", stats.totalFiles))
	} else {
		m.Logger.Info("run complete",
			m.Logger.Args("moved", stats.totalFiles, "size", formatBytes(stats.totalBytes), "categories_skipped", stats.skipped))
	}
	return nil
}

// processCategoryMove handles all extensions for a single category.
func processCategoryMove(m *models.Movelooper, category *models.Category, moved movedSet, batchID string, dryRun, showFiles bool, stats *runStats) {
	files, err := helper.ReadDirectory(category.Source)
	if err != nil {
		m.Logger.Error("failed to read directory", m.Logger.Args("path", category.Source, "error", err.Error()))
		return
	}

	for _, extension := range category.Extensions {
		filteredFiles := filterFilesForExtension(category, files, moved, extension)
		logExtensionResult(m, filteredFiles, category.Name, extension, showFiles)

		stats.totalFiles += len(filteredFiles)
		for _, file := range filteredFiles {
			if info, err := file.Info(); err == nil {
				stats.totalBytes += info.Size()
			}
		}

		if !dryRun && len(filteredFiles) > 0 {
			moveExtension(m, category, filteredFiles, moved, extension, batchID)
		}
	}
}

// moveExtension moves filteredFiles for a single extension, delegating to
// moveAllByExtension when the sentinel "all" is combined with group-by-extension.
func moveExtension(m *models.Movelooper, category *models.Category, files []os.DirEntry, moved movedSet, extension, batchID string) {
	if strings.ToLower(extension) == helper.ExtAll && category.GroupByExtension {
		moveAllByExtension(m, category, files, moved, batchID)
		return
	}

	dirPath := category.Destination
	if category.GroupByExtension {
		dirPath = filepath.Join(category.Destination, extension)
	}
	if err := helper.CreateDirectory(dirPath); err != nil {
		m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
		return
	}
	helper.MoveFiles(helper.MoveContext{Logger: m.Logger, History: m.History}, category, files, extension, batchID)
	for _, file := range files {
		moved.mark(category.Source, file.Name())
	}
}

// moveAllByExtension handles group-by-extension for the "all" sentinel: it groups
// files by their real extension and moves each group into its own subdirectory.
func moveAllByExtension(m *models.Movelooper, category *models.Category, files []os.DirEntry, moved movedSet, batchID string) {
	groups := make(map[string][]os.DirEntry)
	for _, file := range files {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))
		if ext == "" {
			ext = "_no_ext"
		}
		groups[ext] = append(groups[ext], file)
	}

	for ext, group := range groups {
		dirPath := filepath.Join(category.Destination, ext)
		if err := helper.CreateDirectory(dirPath); err != nil {
			m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
			continue
		}
		helper.MoveFiles(helper.MoveContext{Logger: m.Logger, History: m.History}, category, group, ext, batchID)
		for _, file := range group {
			moved.mark(category.Source, file.Name())
		}
	}
}

// formatBytes converts a byte count to a human-readable string (e.g. "1.23 MB").
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGT"[exp])
}

// filterFilesForExtension returns the files that match all criteria for a given extension.
func filterFilesForExtension(category *models.Category, files []os.DirEntry, moved movedSet, extension string) []os.DirEntry {
	var filtered []os.DirEntry
	for _, file := range files {
		if matchesCategory(category, file, moved, extension) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// matchesCategory reports whether a file passes all filters defined by the category.
func matchesCategory(category *models.Category, file os.DirEntry, moved movedSet, extension string) bool {
	if moved.has(category.Source, file.Name()) {
		return false
	}
	if !file.Type().IsRegular() || !helper.HasExtension(file, extension) {
		return false
	}
	if helper.MatchesIgnorePatterns(file.Name(), category.Filter.Ignore) {
		return false
	}
	if !helper.MatchesNameFilters(file.Name(), category.Filter) {
		return false
	}
	info, err := file.Info()
	if err != nil {
		return false
	}
	return helper.MeetsAgeSizeFilters(info, category.Filter)
}

// logExtensionResult logs a summary of files found for an extension.
func logExtensionResult(m *models.Movelooper, files []os.DirEntry, categoryName, extension string, showFiles bool) {
	count := len(files)
	if count == 0 {
		m.Logger.Info(fmt.Sprintf("[%s] No .%s files found", categoryName, extension))
		return
	}
	message := fmt.Sprintf("[%s] %d .%s files to move", categoryName, count, extension)
	if showFiles {
		logArgs := helper.GenerateLogArgs(files, extension)
		if len(logArgs) > 0 {
			m.Logger.Warn(message, m.Logger.Args(logArgs...))
			return
		}
	}
	m.Logger.Warn(message)
}

// preRunHandler handles the necessary configuration before command execution.
// It creates a short-lived Viper instance to read the YAML file, extracts all
// values into typed structs, and discards Viper — the rest of the application
// works exclusively with m.Logger, m.Config, m.Categories, and m.History.
func preRunHandler(m *models.Movelooper, configPath string) error {
	v := viper.New()

	var options []config.ViperOptions
	if configPath != "" {
		// A specific path was provided — use it directly
		dir := filepath.Dir(configPath)
		filename := filepath.Base(configPath)
		ext := filepath.Ext(filename)
		nameWithoutExt := filename[:len(filename)-len(ext)]

		options = []config.ViperOptions{
			config.WithConfigName(nameWithoutExt),
			config.WithConfigType(ext[1:]),
			config.WithConfigPath(dir),
		}
	} else {
		ex, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error getting executable: %v", err)
		}

		options = []config.ViperOptions{
			config.WithConfigName("movelooper"),
			config.WithConfigType("yaml"),
			config.WithConfigPath(filepath.Dir(ex)),
			config.WithConfigPath(filepath.Join(filepath.Dir(ex), "conf")),
		}
	}

	if err := config.InitConfig(v, options...); err != nil {
		if configPath != "" {
			return fmt.Errorf("configuration file not found at '%s'", configPath)
		}
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file")
	}

	logger, closer, err := config.ConfigureLogger(v)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %v", err)
	}
	m.Logger = logger
	m.LogCloser = closer

	m.Config = config.LoadConfig(v)

	categories, err := config.UnmarshalConfig(v)
	if err != nil {
		return err
	}
	m.Categories = categories

	hist, err := history.NewHistory(m.Config.HistoryLimit)
	if err != nil {
		m.Logger.Warn("failed to initialize history tracking", m.Logger.Args("error", err.Error()))
	} else {
		m.History = hist
	}

	validateDirectories(m)

	return nil
}

// validateDirectories warns about source or destination directories that do not exist.
// It does not abort startup — missing directories are reported and skipped at runtime.
func validateDirectories(m *models.Movelooper) {
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		if _, err := os.Stat(cat.Source); os.IsNotExist(err) {
			m.Logger.Warn("source directory does not exist",
				m.Logger.Args("category", cat.Name, "path", cat.Source))
		}
		if _, err := os.Stat(cat.Destination); os.IsNotExist(err) {
			m.Logger.Warn("destination directory does not exist",
				m.Logger.Args("category", cat.Name, "path", cat.Destination))
		}
	}
}
