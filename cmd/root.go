package cmd

import (
	"fmt"
	"movelooper/config"
	"movelooper/models"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "Short description of newMoveLooper",
		Long:  "Long description of newMoveLooper",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			options := []config.ViperOptions{
				config.WithConfigName("movelooper"),
				config.WithConfigType("yaml"),
				config.WithConfigPath("."),
			}

			if m.Viper != nil {
				if err := config.InitConfig(m.Viper, options...); err != nil {
					fmt.Println(err)
				}
			}

			var err error
			if m.Logger == nil {
				m.Logger, err = config.ConfigureLogger()
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
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
		LogLevel:   cmd.PersistentFlags().StringP("log-level", "l", "", "Specify the log level"),
		Output:     cmd.PersistentFlags().StringP("output", "o", "", "Specify the output (console, log or file)"),
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
