package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// ConfigCmd returns the "config" command group
func ConfigCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage movelooper configuration",
	}

	cmd.AddCommand(validateCmd(m))
	cmd.AddCommand(showCmd(m))
	return cmd
}
