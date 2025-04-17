package cmd

import (
	"fmt"
	"movelooper/internal/config"
	"movelooper/internal/helper"
	"movelooper/internal/models"

	"github.com/spf13/cobra"
)

var previewShowFiles bool

// PreviewCmd represents the preview command
func PreviewCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Displays a preview of files to be moved based on configured categories (dry-run)",
		Long: "Displays a preview of files to be moved based on configured categories.\n" +
			"It scans the source directories for each configured category and lists the number of files that match the specified extensions.\n" +
			"This command does not perform any file movement, serving only as a dry-run for verification.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunHandler(cmd, m)
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m.MediaConfig = config.UnmarshalConfig(m)

		for _, category := range m.MediaConfig {
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
					m.Logger.Info(fmt.Sprintf("No %s files to move", extension))
				default:
					message := fmt.Sprintf("%d %s files to move", count, extension)
					if previewShowFiles && len(logArgs) > 0 {
						m.Logger.Warn(message, m.Logger.Args(logArgs...))
					} else {
						m.Logger.Warn(message)
					}
				}
			}
		}
		return nil
	}

	m.Flags = setFlags(cmd)

	bindFlag(cmd, m, "output")
	bindFlag(cmd, m, "log-level")
	bindFlag(cmd, m, "show-caller")

	cmd.Flags().BoolVar(&previewShowFiles, "show-files", false, "Interactive mode for creating a base configuration file")

	return cmd
}
