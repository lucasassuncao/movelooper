package cmd

import (
	"fmt"
	"movelooper/internal/config"
	"movelooper/internal/helper"
	"movelooper/internal/models"
	"path/filepath"

	"github.com/spf13/cobra"
)

var moveShowFiles bool

// MoveCmd represents the move command
func MoveCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "Moves files to their respective destination directories based on configured categories",
		Long: "Moves files to their respective destination directories based on configured categories.\n" +
			"It scans the source directories for each configured category, identifies files matching the specified extensions, and moves them to their corresponding destination directories.\n" +
			"Each file is placed inside a subdirectory named after its extension.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunHandler(cmd, m)
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m.CategoryConfig = config.UnmarshalConfig(m)

		for _, category := range m.CategoryConfig {
			for _, extension := range category.Extensions {
				dirPath := filepath.Join(category.Destination, extension)
				if err := helper.CreateDirectory(dirPath); err != nil {
					m.Logger.Error("failed to create directory",
						m.Logger.Args("directory", dirPath),
						m.Logger.Args("error", err.Error()))
					continue
				}

				files, err := helper.ReadDirectory(category.Source)
				if err != nil {
					m.Logger.Error("failed to read source directory",
						m.Logger.Args("path", category.Source),
						m.Logger.Args("error", err.Error()),
					)
					continue
				}

				count := helper.ValidateFiles(files, extension)
				logArgs := helper.GenerateLogArgs(files, extension) // This contains pairs of "name", fileName

				if count == 0 {
					m.Logger.Info("no files to move",
						m.Logger.Args("extension", extension),
						m.Logger.Args("source_directory", category.Source),
					)
				} else {
					baseMessage := "files to move"
					args := []interface{}{
						"count", count,
						"extension", extension,
						"source_directory", category.Source,
						"destination_directory", dirPath,
					}
					if moveShowFiles && len(logArgs) > 0 {
						// logArgs is already a slice of interface{}, e.g., ["name", "file1.txt", "name", "file2.txt"]
						args = append(args, logArgs...)
						m.Logger.Info(baseMessage, m.Logger.Args(args...))
					} else {
						m.Logger.Info(baseMessage, m.Logger.Args(args...))
					}
				}

				helper.MoveFiles(m, category, files, extension)
			}
		}
		return nil
	}

	m.Flags = setFlags(cmd)

	bindFlag(cmd, m, "output")
	bindFlag(cmd, m, "log-level")
	bindFlag(cmd, m, "show-caller")

	cmd.Flags().BoolVar(&moveShowFiles, "show-files", false, "Interactive mode for creating a base configuration file")

	return cmd
}
