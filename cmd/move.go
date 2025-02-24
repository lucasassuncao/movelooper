package cmd

import (
	"fmt"
	"movelooper/models"
	"path/filepath"

	"github.com/spf13/cobra"
)

func MoveCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "Moves files to their respective destination directories based on configured categories",
		Long: "Moves files to their respective destination directories based on configured categories\n" +
			"It scans the source directories for each configured category, identifies files matching the specified extensions, and moves them to their corresponding destination directories\n" +
			"Each file is placed inside a subdirectory named after its extension",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		m.Logger.Info("Starting newMoveLooper")
		m.MediaConfig.AllCategories = getCategories(m.Viper)

		for _, category := range m.MediaConfig.AllCategories {
			m.MediaConfig.Category = category

			m.MediaConfig.Extensions = m.Viper.GetStringSlice(fmt.Sprintf("categories.%s.extensions", category))
			m.MediaConfig.Source = m.Viper.GetString(fmt.Sprintf("categories.%s.source", category))
			m.MediaConfig.Destination = m.Viper.GetString(fmt.Sprintf("categories.%s.destination", category))

			for _, extension := range m.MediaConfig.Extensions {
				createDirectory(m, filepath.Join(m.MediaConfig.Destination, extension))

				files := readDirectory(m, m.MediaConfig.Source)
				count := validateFiles(files, extension)

				switch count {
				case 0:
					m.Logger.Info(fmt.Sprintf("No .%s file(s) to move", extension))
					moveFile(m, files, extension)
				case 1:
					m.Logger.Warn(fmt.Sprintf("%d file .%s to move", count, extension))
					moveFile(m, files, extension)
				default:
					m.Logger.Warn(fmt.Sprintf("%d files .%s to move", count, extension))
					moveFile(m, files, extension)
				}
			}
		}
		return nil
	}
	return cmd
}
