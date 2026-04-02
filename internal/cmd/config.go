package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ConfigCmd returns the "config" command group
func ConfigCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage movelooper configuration",
	}

	cmd.AddCommand(validateCmd(m))
	return cmd
}

func validateCmd(m *models.Movelooper) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file without moving any files",
		RunE: func(cmd *cobra.Command, args []string) error {
			categories, err := config.UnmarshalConfig(m)
			if err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			pterm.Success.Println("Configuration is valid")
			pterm.Println()

			for _, cat := range categories {
				pterm.Printf("  Category : %s\n", pterm.Cyan(cat.Name))
				pterm.Printf("  Source   : %s\n", pterm.Yellow(cat.Source))
				pterm.Printf("  Dest     : %s\n", pterm.Yellow(cat.Destination))
				pterm.Printf("  Exts     : %s\n", pterm.Green(strings.Join(cat.Extensions, ", ")))

				if cat.Filter.Regex != "" {
					pterm.Printf("  Regex    : %s\n", pterm.Magenta(cat.Filter.Regex))
				}
				if cat.Filter.Glob != "" {
					pterm.Printf("  Glob     : %s\n", pterm.Magenta(cat.Filter.Glob))
				}
				if len(cat.Filter.Ignore) > 0 {
					pterm.Printf("  Ignore   : %s\n", pterm.Red(strings.Join(cat.Filter.Ignore, ", ")))
				}
				if cat.Filter.MinAge > 0 {
					pterm.Printf("  Min Age  : %s\n", pterm.Yellow(cat.Filter.MinAge.String()))
				}
				if cat.Filter.MaxAge > 0 {
					pterm.Printf("  Max Age  : %s\n", pterm.Yellow(cat.Filter.MaxAge.String()))
				}
				if cat.Filter.MinSize != "" {
					pterm.Printf("  Min Size : %s\n", pterm.Yellow(cat.Filter.MinSize))
				}
				if cat.Filter.MaxSize != "" {
					pterm.Printf("  Max Size : %s\n", pterm.Yellow(cat.Filter.MaxSize))
				}
				if cat.ConflictStrategy != "" {
					pterm.Printf("  Strategy : %s\n", cat.ConflictStrategy)
				}

				pterm.Println()
			}

			pterm.Printf("  %d %s loaded\n",
				len(categories),
				pluralize("category", "categories", len(categories)),
			)

			return nil
		},
	}
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
