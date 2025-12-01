// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	var (
		dryRun    bool
		showFiles bool
	)

	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "movelooper is a CLI tool for organizing and moving files",
		Long: `movelooper organizes and moves files from source directories to destination directories,
based on configurable categories.

By default, it runs the move operation automatically.
Use -p / --preview / --dry-run for a dry-run preview, and --show-files to display filenames.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return preRunHandler(m, configPath)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			m.Categories = config.UnmarshalConfig(m)

			for _, category := range m.Categories {
				for _, extension := range category.Extensions {
					files, err := helper.ReadDirectory(category.Source)
					if err != nil {
						m.Logger.Error("failed to read directory",
							m.Logger.Args("path", category.Source),
							m.Logger.Args("error", err.Error()),
						)
						continue
					}

					count := helper.ValidateFiles(files, extension)
					logArgs := helper.GenerateLogArgs(files, extension)

					switch count {
					case 0:
						m.Logger.Info(fmt.Sprintf("No .%s files found", extension))
					default:
						var message string
						if category.Regex != "" && !category.UseExtensionSubfolder {
							message = fmt.Sprintf("%d files from category %s to move", count, category.Name)
						} else {
							message = fmt.Sprintf("%d .%s files to move", count, extension)
						}

						if showFiles && len(logArgs) > 0 {
							m.Logger.Warn(message, m.Logger.Args(logArgs...))
						} else {
							m.Logger.Warn(message)
						}
					}

					// Only move files if not in dry-run mode
					if !dryRun {
						dirPath := filepath.Join(category.Destination, extension)
						if err := helper.CreateDirectory(dirPath); err != nil {
							m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
							continue
						}
						helper.MoveFiles(m, category, files, extension)
					}
				}
			}

			if dryRun {
				m.Logger.Info("Dry-run complete (no files were moved).")
			}

			return nil
		},
	}

	cmd.Flags().StringP("config", "c", "", "Path to configuration file (e.g., /path/to/movelooper.yaml)")
	cmd.Flags().BoolVarP(&dryRun, "preview", "p", false, "Run in dry-run (preview) mode without moving files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Alias for --preview")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")

	// Add subcommands
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(WatchCmd(m))

	return cmd
}

// preRunHandler handles the necessary configuration before command execution
func preRunHandler(m *models.Movelooper, configPath string) error {
	var options []config.ViperOptions

	if configPath != "" {
		// Se um caminho específico foi fornecido, use-o
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

	logger, err := config.ConfigureLogger(m.Viper)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %v", err)
	}

	m.Logger = logger

	return nil
}
