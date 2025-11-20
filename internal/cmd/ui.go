package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/ui"
	"github.com/spf13/cobra"
)

// UiCmd defines the "ui" command to launch the graphical interface
func UiCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the graphical user interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			if err := preRunHandler(m, configPath); err != nil {
				return err
			}
			
			ui.StartUI(m)
			return nil
		},
	}

	cmd.Flags().StringP("config", "c", "", "Path to configuration file")
	return cmd
}
