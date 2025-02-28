package cmd

import (
	"fmt"
	"movelooper/internal/config"
	"movelooper/internal/models"
	"path/filepath"

	"github.com/spf13/cobra"
)

// MoveCmd represents the move command
func MoveCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "Moves files to their respective destination directories based on configured categories",
		Long: "Moves files to their respective destination directories based on configured categories\n" +
			"It scans the source directories for each configured category, identifies files matching the specified extensions, and moves them to their corresponding destination directories\n" +
			"Each file is placed inside a subdirectory named after its extension",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if m.Logger == nil {
			return fmt.Errorf("logger is not initialized")
		}

		m.Logger.Info("Starting move mode")

		m.MediaConfig = config.UnmarshalConfig(m)

		for _, category := range m.MediaConfig {
			for _, extension := range category.Extensions {
				createDirectory(m, filepath.Join(category.Destination, extension))

				files := readDirectory(m, category.Source)
				count := validateFiles(files, extension)

				switch count {
				case 0:
					m.Logger.Info(fmt.Sprintf("No %s file(s) to move", extension))
					moveFile(m, category, files, extension)
				case 1:
					m.Logger.Warn(fmt.Sprintf("%d file %s to move", count, extension))
					moveFile(m, category, files, extension)
				default:
					m.Logger.Warn(fmt.Sprintf("%d files %s to move", count, extension))
					moveFile(m, category, files, extension)
				}
			}
		}
		return nil
	}
	return cmd
}
