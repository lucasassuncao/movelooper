// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
)

// MoveCmd represents the move command
func MoveCmd(m *models.Movelooper) *cobra.Command {
	var dryRun bool
	var showFiles bool

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Moves files based on configuration (default command)",
		Long: `By default, 'movelooper' runs the move operation. 
You can use -p or --preview (or --dry-run) to perform a dry-run preview without moving files.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunHandler(cmd, m)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			m.CategoryConfig = config.UnmarshalConfig(m)

			for _, category := range m.CategoryConfig {
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
						message := fmt.Sprintf("%d .%s files to move", count, extension)
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

	m.Flags = setFlags(cmd)

	bindFlag(cmd, m, "output")
	bindFlag(cmd, m, "log-level")
	bindFlag(cmd, m, "show-caller")

	cmd.Flags().BoolVarP(&dryRun, "preview", "p", false, "Run in dry-run (preview) mode")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Alias for --preview")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "Show list of files that would be moved")

	return cmd
}
