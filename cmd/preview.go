package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"movelooper/models"
)

func PreviewCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Displays a preview of files to be moved based on configured categories",
		Long: "Displays a preview of files to be moved based on configured categories\n" +
			"It scans the source directories for each configured category and lists the number of files that match the specified extensions\n" +
			"This command does not perform any file movement, serving only as a dry-run for verification",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m.Logger.Info("Starting newMoveLooper")
		m.MediaConfig.AllCategories = getCategories(m.Viper)

		for _, category := range m.MediaConfig.AllCategories {
			m.MediaConfig.Category = category

			m.MediaConfig.Extensions = m.Viper.GetStringSlice(fmt.Sprintf("categories.%s.extensions", category))
			m.MediaConfig.Source = m.Viper.GetString(fmt.Sprintf("categories.%s.source", category))

			for _, extension := range m.MediaConfig.Extensions {
				files := readDirectory(m, m.MediaConfig.Source)
				count := validateFiles(files, extension)

				switch count {
				case 0:
					m.Logger.Info(fmt.Sprintf("No .%s file(s) to move", extension))
				case 1:
					m.Logger.Warn(fmt.Sprintf("%d file .%s to move", count, extension))
				default:
					m.Logger.Warn(fmt.Sprintf("%d files .%s to move", count, extension))
				}
			}
		}
		return nil
	}
	return cmd
}
