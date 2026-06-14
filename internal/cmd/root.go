// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"errors"
	"fmt"

	"github.com/lucasassuncao/movelooper/internal/config"
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
			opts := MoveOptions{
				DryRun:          dryRun,
				ShowFiles:       showFiles,
				CategoryFilter:  categoryFilter,
				IncludeDisabled: includeDisabled,
			}
			return runMove(cmd.Context(), m, opts)
		},
	}

	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file (e.g., /path/to/movelooper.yaml)")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview mode! It shows what would be moved without moving files")
	cmd.PersistentFlags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to process (default: all)")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include categories with enabled: false")

	// Add subcommands
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(EditCmd())
	cmd.AddCommand(WatchCmd(m))
	cmd.AddCommand(UndoCmd(m))
	cmd.AddCommand(SelfUpdateCmd(version))
	cmd.AddCommand(ShowCmd)
	cmd.AddCommand(GenerateCmd)

	return cmd
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
			return fmt.Errorf("configuration file not found at %q: %w", configPath, err)
		}
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file: %w", err)
	}
	return err
}
