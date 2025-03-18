package cmd

import (
	"fmt"
	"log"
	"movelooper/internal/config"
	"movelooper/internal/models"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "movelooper",
		Short: "movelooper is a CLI tool for organizing and moving files",
		Long:  "movelooper is a CLI tool for organizing and moving files from source directories to destination directories, based on configurable categories",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ex, err := os.Executable()
			if err != nil {
				log.Fatalf("error getting executable: %v", err)
				return
			}

			options := []config.ViperOptions{
				config.WithConfigName("movelooper"),
				config.WithConfigType("yaml"),
				config.WithConfigPath(filepath.Dir(ex)),
				config.WithConfigPath(filepath.Join(filepath.Dir(ex), "conf")),
			}

			err = config.InitConfig(m.Viper, options...)
			if err != nil {
				fmt.Printf("failed to initialize config: %v\nlaunching baseconfig to create a new config file then run the app again", err)
				time.Sleep(5 * time.Second)
				cmd := BaseConfigCmd(m)
				cmd.SetArgs([]string{"--interactive"})
				cmd.Execute()
				_ = config.InitConfig(m.Viper, options...)
			}

			logger, err := config.ConfigureLogger(m.Viper)
			if err != nil {
				fmt.Printf("failed to configure logger: %v\n", err)
				return
			}

			m.Logger = logger

			if m.Flags == nil {
				m.Logger.Error("error configuring flags")
			}

			checkFlags(cmd, m, m.Flags, "output")
			checkFlags(cmd, m, m.Flags, "show-caller")
			checkFlags(cmd, m, m.Flags, "log-level")
		},
	}

	m.Flags = setPersistentFlags(cmd)

	bindPersistentFlag(cmd, m, "output")
	bindPersistentFlag(cmd, m, "log-level")
	bindPersistentFlag(cmd, m, "show-caller")

	cmd.AddCommand(PreviewCmd(m))
	cmd.AddCommand(MoveCmd(m))
	cmd.AddCommand(BaseConfigCmd(m))

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
func bindPersistentFlag(cmd *cobra.Command, m *models.Movelooper, flagName string) {
	// Bind the flag to a Viper key and handle any binding errors
	err := m.Viper.BindPFlag(fmt.Sprintf("configuration.%s", flagName), cmd.PersistentFlags().Lookup(flagName))
	if err != nil {
		m.Logger.Error("error binding flag", m.Logger.Args("flag", flagName, "error", err))
	}
}

// checkFlags ensures that the flags are set correctly, either from the command-line or from the Viper configuration
func checkFlags(cmd *cobra.Command, m *models.Movelooper, flags *models.PersistentFlags, flagName string) {
	// If the flag was not changed by the user, check Viper and set it if needed
	if !cmd.PersistentFlags().Changed(flagName) && m.Viper.IsSet(fmt.Sprintf("configuration.%s", flagName)) {
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
