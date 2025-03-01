package cmd

import (
	"fmt"
	"movelooper/internal/models"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// BaseConfigCmd generates a base configuration file
func BaseConfigCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseconfig",
		Short: "Generates a base configuration file",
		Long: "Generates a base configuration file in the application directory with predefined categories.\n" +
			"This file can be customized to define category names, file extensions, source directories, and destination paths.\n" +
			"If the base configuration file already exists, it will not be overwritten.",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if m.Logger == nil {
			return fmt.Errorf("logger is not initialized")
		}

		m.Logger.Info("Creating a base configuration file")

		ex, err := os.Executable()
		if err != nil {
			m.Logger.Error("error getting executable", m.Logger.Args("error", err))
			return err
		}

		path := filepath.Join(filepath.Dir(ex), "conf", "base")

		err = createDirectory(path)
		if err != nil {
			m.Logger.Error("error creating directory for base config", m.Logger.Args("error", err))
		}

		err = models.NewConfig(path)
		if err != nil {
			m.Logger.Error("error creating base configuration file", m.Logger.Args("error", err))
		}

		m.Logger.Info("Base configuration file created", m.Logger.Args("path", path))

		return nil
	}
	return cmd
}
