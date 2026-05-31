package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// validateCmd returns the "config validate" command that checks if the configuration file is valid without moving any files
func validateCmd(m *models.Movelooper) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file without moving any files",
		RunE: func(cmd *cobra.Command, args []string) error {
			pterm.Success.Println("Configuration is valid")
			pterm.Println()

			for _, cat := range m.Categories {
				printCategorySummary(*cat)
				pterm.Println()
			}

			pterm.Printf("  %d %s loaded\n",
				len(m.Categories),
				pluralize("category", "categories", len(m.Categories)),
			)

			return nil
		},
	}
}

// pluralize returns the singular or plural form of a word based on the count
func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
