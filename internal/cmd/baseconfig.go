package cmd

import (
	"fmt"
	"movelooper/internal/helper"
	"movelooper/internal/models"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var interactive bool

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
		ex, err := os.Executable()
		if err != nil {
			fmt.Printf("error getting executable: %v\n", err)
			return err
		}

		configPath := filepath.Join(filepath.Dir(ex), "conf")
		baseconfigPath := filepath.Join(filepath.Dir(ex), "conf", "base")

		err = helper.CreateDirectory(baseconfigPath)
		if err != nil {
			fmt.Printf("error creating directory for base config: %v\n", err)
		}

		var options = []models.ConfigOption{}

		if interactive {
			options = append(options, models.WithOutput())
			options = append(options, models.WithLogFile())
			options = append(options, models.WithLogLevel())
			options = append(options, models.WithShowCaller())
			options = append(options, models.WithCategory())

		}

		err = models.NewConfig(configPath, baseconfigPath, interactive, options...)
		if err != nil {
			fmt.Printf("error creating base configuration file: %v\n", err)
		}

		return nil
	}

	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive mode for creating a base configuration file")

	return cmd
}
