package cmd

import (
	"fmt"
	"movelooper/internal/config"
	"movelooper/internal/models"

	"github.com/spf13/cobra"
)

// PreviewCmd represents the preview command
func PreviewCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Displays a preview of files to be moved based on configured categories (dry-run)",
		Long: "Displays a preview of files to be moved based on configured categories\n" +
			"It scans the source directories for each configured category and lists the number of files that match the specified extensions\n" +
			"This command does not perform any file movement, serving only as a dry-run for verification",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if m.Logger == nil {
			return fmt.Errorf("logger is not initialized")
		}

		m.Logger.Info("Starting preview mode")

		m.MediaConfig = config.UnmarshalConfig(m)

		for _, category := range m.MediaConfig {
			for _, extension := range category.Extensions {

				files := readDirectory(m, category.Source)
				count := validateFiles(files, extension)

				switch count {
				case 0:
					m.Logger.Info(fmt.Sprintf("No %s file(s) to move", extension))
				case 1:
					m.Logger.Warn(fmt.Sprintf("%d file %s to move", count, extension))
				default:
					m.Logger.Warn(fmt.Sprintf("%d files %s to move", count, extension))
				}
			}
		}
		return nil
	}
	return cmd
}
