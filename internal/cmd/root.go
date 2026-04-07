// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
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

func (s movedSet) mark(dir, name string) { s[filepath.Join(dir, name)] = true }
func (s movedSet) has(dir, name string) bool { return s[filepath.Join(dir, name)] }

// runMove executes the default move operation across all configured categories.
func runMove(m *models.Movelooper, dryRun, showFiles bool) error {
	categories, err := config.UnmarshalConfig(m)
	if err != nil {
		return err
	}
	m.Categories = categories

	batchID := history.NewBatchID()
	moved := make(movedSet)

	for _, category := range m.Categories {
		processCategoryMove(m, category, moved, batchID, dryRun, showFiles)
	}

	if dryRun {
		m.Logger.Info("dry-run complete, no files were moved")
	}
	return nil
}

// processCategoryMove handles all extensions for a single category.
func processCategoryMove(m *models.Movelooper, category *models.Category, moved movedSet, batchID string, dryRun, showFiles bool) {
	files, err := helper.ReadDirectory(category.Source)
	if err != nil {
		m.Logger.Error("failed to read directory", m.Logger.Args("path", category.Source, "error", err.Error()))
		return
	}

	for _, extension := range category.Extensions {
		filteredFiles := filterFilesForExtension(category, files, moved, extension)
		logExtensionResult(m, filteredFiles, category.Name, extension, showFiles)

		if !dryRun && len(filteredFiles) > 0 {
			dirPath := filepath.Join(category.Destination, extension)
			if err := helper.CreateDirectory(dirPath); err != nil {
				m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
			}
			helper.MoveFiles(m, category, filteredFiles, extension, batchID)
			for _, file := range filteredFiles {
				moved.mark(category.Source, file.Name())
			}
		}
	}
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

// preRunHandler handles the necessary configuration before command execution
func preRunHandler(m *models.Movelooper, configPath string) error {
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

	err := config.InitConfig(m.Viper, options...)
	if err != nil {
		if configPath != "" {
			return fmt.Errorf("configuration file not found at '%s'", configPath)
		}
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file")
	}

	logger, closer, err := config.ConfigureLogger(m.Viper)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %v", err)
	}

	m.Logger = logger
	m.LogCloser = closer

	historyLimit := m.Viper.GetInt("configuration.history-limit")
	hist, err := history.NewHistory(historyLimit)
	if err != nil {
		// Log warning but don't fail app if history fails
		m.Logger.Warn("failed to initialize history tracking", m.Logger.Args("error", err.Error()))
	} else {
		m.History = hist
	}

	return nil
}
