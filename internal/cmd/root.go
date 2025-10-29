// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	var (
		dryRun    bool
		showFiles bool
	)

	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "movelooper is a CLI tool for organizing and moving files",
		Long: `movelooper organizes and moves files from source directories to destination directories,
based on configurable categories.

By default, it runs the move command automatically.
Use -p / --preview / --dry-run for a dry-run preview, and --show-files to display filenames.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunHandler(cmd, m)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reuse MoveCmd behavior
			moveCmd := MoveCmd(m)

			// Transfer root flags to MoveCmd
			moveArgs := []string{}
			if dryRun {
				moveArgs = append(moveArgs, "--dry-run")
			}
			if showFiles {
				moveArgs = append(moveArgs, "--show-files")
			}

			// Preserve any extra args the user may have passed
			moveArgs = append(moveArgs, args...)
			moveCmd.SetArgs(moveArgs)

			return moveCmd.Execute()
		},
	}

	m.Flags = setFlags(cmd)

	bindFlag(cmd, m, "output")
	bindFlag(cmd, m, "log-level")
	bindFlag(cmd, m, "show-caller")

	// Add subcommands
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(MoveCmd(m))

	// Register move-related flags here too
	cmd.Flags().BoolVarP(&dryRun, "preview", "p", false, "Run in dry-run (preview) mode without moving files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Alias for --preview")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "Show list of individual files detected")

	return cmd
}

// setFlags sets the flags for a Cobra command
func setFlags(cmd *cobra.Command) *models.Flags {
	return &models.Flags{
		ShowCaller: cmd.Flags().Bool("show-caller", false, "Show caller information"),
		LogLevel:   cmd.Flags().StringP("log-level", "l", "", "Specify the log level (trace, debug, info, warn/warning, error, fatal)"),
		Output:     cmd.Flags().StringP("output", "o", "", "Specify the output (console, log/file or both)"),
	}
}

// bindFlag links a CLI flag to a Viper key to enable configuration file support
func bindFlag(cmd *cobra.Command, m *models.Movelooper, flagName string) {
	// Bind the flag to a Viper key and handle any binding errors
	err := m.Viper.BindPFlag(fmt.Sprintf("configuration.%s", flagName), cmd.Flags().Lookup(flagName))
	if err != nil {
		m.Logger.Error("error binding flag", m.Logger.Args("flag", flagName, "error", err))
	}
}

// checkFlags ensures that the flags are set correctly, either from the command-line or from the Viper configuration
func checkFlags(cmd *cobra.Command, m *models.Movelooper, flags *models.Flags, flagName string) {
	// If the flag was not changed by the user, check Viper and set it if needed
	if !cmd.Flags().Changed(flagName) && m.Viper.IsSet(fmt.Sprintf("configuration.%s", flagName)) {
		switch flagName {
		case "output":
			*flags.Output = m.Viper.GetString(fmt.Sprintf("configuration.%s", flagName))
		case "log-level":
			*flags.LogLevel = m.Viper.GetString(fmt.Sprintf("configuration.%s", flagName))
		case "show-caller":
			*flags.ShowCaller = m.Viper.GetBool(fmt.Sprintf("configuration.%s", flagName))
		}
	}
}

// preRunHandler handles the necessary configuration before command execution
func preRunHandler(cmd *cobra.Command, m *models.Movelooper) error {
	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting executable: %v", err)
	}

	options := []config.ViperOptions{
		config.WithConfigName("movelooper"),
		config.WithConfigType("yaml"),
		config.WithConfigPath(filepath.Dir(ex)),
		config.WithConfigPath(filepath.Join(filepath.Dir(ex), "conf")),
	}

	err = config.InitConfig(m.Viper, options...)
	if err != nil {
		return fmt.Errorf("configuration file not found\n\nPlease run 'movelooper init' to create a configuration file")
	}

	logger, err := config.ConfigureLogger(m.Viper)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %v", err)
	}

	m.Logger = logger

	if m.Flags == nil {
		m.Logger.Error("error configuring flags")
	}

	checkFlags(cmd, m, m.Flags, "output")
	checkFlags(cmd, m, m.Flags, "show-caller")
	checkFlags(cmd, m, m.Flags, "log-level")

	return nil
}
