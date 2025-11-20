package core

import (
	"fmt"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// RunOnce executes the file organization process a single time
func RunOnce(m *models.Movelooper, dryRun bool, showFiles bool) error {
	// Refresh config in case it changed (useful for UI)
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
}
