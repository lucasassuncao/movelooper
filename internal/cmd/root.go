// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"errors"
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
		dryRun          bool
		showFiles       bool
		categoryFilter  string
		includeDisabled bool
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
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			return preRunHandler(m, configPath)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if m.LogCloser != nil {
				return m.LogCloser.Close()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(m, dryRun, showFiles, categoryFilter, includeDisabled)
		},
	}

	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (e.g., /path/to/movelooper.yaml)")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview mode! It shows what would be moved without moving files")
	cmd.PersistentFlags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to process (default: all)")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include categories with enabled: false")

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
func runMove(m *models.Movelooper, dryRun, showFiles bool, categoryFilter string, includeDisabled bool) error {
	names := parseCategoryNames(categoryFilter)
	categories, err := filterCategories(m.Categories, names, includeDisabled, m.Logger)
	if err != nil {
		return err
	}

	batchID := history.NewBatchID()
	moved := make(movedSet)
	var stats runStats

	for _, category := range categories {
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

// hookAfterVars carries the post-move stats needed for "after" hook env vars.
type hookAfterVars struct {
	moved   int
	failed  int
	batchID string
}

// hookEnv builds the environment variable map to inject into a hook process.
// afterVars is non-nil only for "after" hooks.
func hookEnv(category *models.Category, dryRun bool, after *hookAfterVars) map[string]string {
	action := category.Destination.Action
	if action == "" {
		action = "move"
	}
	dry := "false"
	if dryRun {
		dry = "true"
	}
	env := map[string]string{
		"ML_CATEGORY":    category.Name,
		"ML_SOURCE_PATH": category.Source.Path,
		"ML_DEST_PATH":   category.Destination.Path,
		"ML_DRY_RUN":     dry,
		"ML_ACTION":      action,
	}
	if after != nil {
		env["ML_FILES_MOVED"] = fmt.Sprintf("%d", after.moved)
		env["ML_FILES_SKIPPED"] = "0"
		env["ML_FILES_FAILED"] = fmt.Sprintf("%d", after.failed)
		env["ML_BATCH_ID"] = after.batchID
	}
	return env
}

// processCategoryMove handles all extensions for a single category.
func processCategoryMove(m *models.Movelooper, category *models.Category, moved movedSet, batchID string, dryRun, showFiles bool, stats *runStats) {
	if category.Hooks != nil && category.Hooks.Before != nil {
		env := hookEnv(category, dryRun, nil)
		if err := helper.RunHook(category.Hooks.Before, m.Logger, env); err != nil {
			m.Logger.Warn("before hook failed, skipping category",
				m.Logger.Args("category", category.Name, "error", err.Error()))
			stats.skipped++
			return
		}
	}

	files, err := helper.ReadDirectory(category.Source.Path)
	if err != nil {
		m.Logger.Error("failed to read directory", m.Logger.Args("path", category.Source.Path, "error", err.Error()))
		return
	}

	var totalMoved, totalFailed int
	for _, extension := range category.Source.Extensions {
		filteredFiles := filterFilesForExtension(category, files, moved, extension)
		logExtensionResult(m, filteredFiles, category.Name, extension, showFiles)

		stats.totalFiles += len(filteredFiles)
		for _, file := range filteredFiles {
			if info, err := file.Info(); err == nil {
				stats.totalBytes += info.Size()
			} else {
				m.Logger.Warn("could not stat file for size accounting", m.Logger.Args("file", file.Name(), "error", err.Error()))
			}
		}

		if !dryRun && len(filteredFiles) > 0 {
			names := moveExtensionWithResult(m, category, filteredFiles, moved, extension, batchID)
			totalMoved += len(names)
			totalFailed += len(filteredFiles) - len(names)
		}
	}

	if category.Hooks != nil && category.Hooks.After != nil {
		env := hookEnv(category, dryRun, &hookAfterVars{
			moved:   totalMoved,
			failed:  totalFailed,
			batchID: batchID,
		})
		if err := helper.RunHook(category.Hooks.After, m.Logger, env); err != nil {
			m.Logger.Warn("after hook failed",
				m.Logger.Args("category", category.Name, "error", err.Error()))
		}
	}
}

// moveExtensionWithResult moves filteredFiles for a single extension and returns moved file names.
func moveExtensionWithResult(m *models.Movelooper, category *models.Category, files []os.DirEntry, moved movedSet, extension, batchID string) []string {
	movedNames := helper.MoveFiles(helper.MoveContext{Logger: m.Logger, History: m.History}, category, files, extension, batchID)
	for _, name := range movedNames {
		moved.mark(category.Source.Path, name)
	}
	return movedNames
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
	const prefixes = "KMGTPE"
	if exp >= len(prefixes) {
		exp = len(prefixes) - 1
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), prefixes[exp])
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
	if moved.has(category.Source.Path, file.Name()) {
		return false
	}
	if !file.Type().IsRegular() || !helper.HasExtension(file, extension) {
		return false
	}
	info, err := file.Info()
	if err != nil {
		return false
	}
	return helper.MatchesFilter(category.Source.Filter, file.Name(), info)
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
func preRunHandler(m *models.Movelooper, configPath string) (retErr error) {
	defer func() {
		if retErr != nil && m.LogCloser != nil {
			m.LogCloser.Close()
			m.LogCloser = nil
		}
	}()

	err := config.NewAppBuilder(m, configPath).
		ResolveConfig().
		ConfigureLogger().
		LoadConfig().
		LoadCategories().
		InitHistory().
		ValidateDirectories().
		Build()

	if errors.Is(err, config.ErrConfigNotFound) {
		if configPath != "" {
			return fmt.Errorf("configuration file not found at '%s'", configPath)
		}
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file")
	}
	return err
}
