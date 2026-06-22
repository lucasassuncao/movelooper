// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
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
		Use:           "movelooper",
		Short:         "movelooper is a CLI tool for organizing and moving files",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview mode! It shows what would be moved without moving files")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to process (default: all)")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include categories with enabled: false")

	cmd.AddGroup(
		&cobra.Group{ID: "ops", Title: "File Operation Commands"},
		&cobra.Group{ID: "config", Title: "Configuration Commands"},
		&cobra.Group{ID: "utils", Title: "Utility Commands"},
	)

	watchCmd := WatchCmd(m)
	watchCmd.GroupID = "ops"
	undoCmd := UndoCmd(m)
	undoCmd.GroupID = "ops"

	editCmd := EditCmd()
	editCmd.GroupID = "config"
	validateCmd := ValidateCmd()
	validateCmd.GroupID = "config"
	configCmd := ConfigCmd()
	configCmd.GroupID = "config"

	selfUpdateCmd := SelfUpdateCmd(version)
	selfUpdateCmd.GroupID = "utils"
	showCmd := ShowCmd()
	showCmd.GroupID = "utils"

	GenerateCmd.GroupID = "utils"
	cmd.AddCommand(watchCmd, undoCmd, editCmd, validateCmd, configCmd, selfUpdateCmd, showCmd, GenerateCmd)

	cmd.CompletionOptions.HiddenDefaultCmd = true
	cmd.SetHelpCommand(&cobra.Command{Hidden: true, GroupID: "utils"})

	return cmd
}

// preRunHandler handles the necessary configuration before command execution.
func preRunHandler(m *models.Movelooper, configPath string) error {
	return config.NewApp(m, configPath,
		config.WithLogger(),
		config.WithConfig(),
		config.WithCategories(),
		config.WithHistory(),
		config.WithValidateDirs(),
	)
}
