package cmd

import (
	"fmt"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/spf13/cobra"
)

// ConfigCmd returns the "config" command for configuration utilities.
func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "config",
		Short:             "Show the resolved configuration file path",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Print the absolute path of the configuration file that movelooper would use.

Respects --config when provided; otherwise searches the default locations:
  ~/.movelooper/conf/movelooper.yaml
  <executable-dir>/movelooper.yaml
  <executable-dir>/conf/movelooper.yaml`,
		Example: `  movelooper config
  movelooper --config /path/to/movelooper.yaml config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			path, err := config.ResolveConfigPath(configPath)
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
	return cmd
}
