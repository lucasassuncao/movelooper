package cmd

import (
	"fmt"
	"log"
	"movelooper/internal/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "movelooper is a CLI tool for organizing and moving files",
		Long:  "movelooper is a CLI tool for organizing and moving files from source directories to destination directories, based on configurable categories",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if m.Flags == nil {
				log.Fatalf("error configuring flags")
			}

			checkFlags(cmd, m.Flags, "output")
			checkFlags(cmd, m.Flags, "show-caller")
			checkFlags(cmd, m.Flags, "log-level")
		},
	}

	m.Flags = setPersistentFlags(cmd)

	bindPersistentFlag(cmd, "output")
	bindPersistentFlag(cmd, "log-level")
	bindPersistentFlag(cmd, "show-caller")

	cmd.AddCommand(PreviewCmd(m))
	cmd.AddCommand(MoveCmd(m))

	return cmd
}

// setPersistentFlags sets the persistent flags for a Cobra command, which are flags
// that are available to the command and all of its subcommands.
func setPersistentFlags(cmd *cobra.Command) *models.PersistentFlags {
	return &models.PersistentFlags{
		ShowCaller: cmd.PersistentFlags().Bool("show-caller", false, "Show caller information"),
		LogLevel:   cmd.PersistentFlags().StringP("log-level", "l", "", "Specify the log level (trace, debug, info, warn/warning, error, fatal)"),
		Output:     cmd.PersistentFlags().StringP("output", "o", "", "Specify the output (console, log/file or both)"),
	}
}

// bindPersistentFlag links a CLI flag to a Viper key to enable configuration file support
func bindPersistentFlag(cmd *cobra.Command, flagName string) {
	// Bind the flag to a Viper key and handle any binding errors
	err := viper.BindPFlag(fmt.Sprintf("configuration.%s", flagName), cmd.PersistentFlags().Lookup(flagName))
	if err != nil {
		_ = fmt.Errorf("error binding flag %s: %w", flagName, err)
	}
}

// checkFlags ensures that the flags are set correctly, either from the command-line or from the Viper configuration
func checkFlags(cmd *cobra.Command, flags *models.PersistentFlags, flagName string) {
	// If the flag was not changed by the user, check Viper and set it if needed
	if !cmd.PersistentFlags().Changed(flagName) && viper.IsSet(fmt.Sprintf("configuration.%s", flagName)) {
		switch flagName {
		case "output":
			*flags.Output = viper.GetString(fmt.Sprintf("configuration.%s", flagName))
		case "log-level":
			*flags.LogLevel = viper.GetString(fmt.Sprintf("configuration.%s", flagName))
		case "show-caller":
			*flags.ShowCaller = viper.GetBool(fmt.Sprintf("configuration.%s", flagName))
		}
	}
}
