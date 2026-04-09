package cmd

import (
	"fmt"
	"strings"

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
	cmd.AddCommand(showCmd(m))
	return cmd
}

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

func showCmd(m *models.Movelooper) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the active configuration with all defaults resolved",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := m.Config

			pterm.DefaultSection.Println("Configuration")
			pterm.Printf("  %-20s %s\n", "output:", cfg.Output)
			pterm.Printf("  %-20s %s\n", "log-file:", orDash(cfg.LogFile))
			pterm.Printf("  %-20s %s\n", "log-level:", cfg.LogLevel)
			pterm.Printf("  %-20s %v\n", "show-caller:", cfg.ShowCaller)
			pterm.Printf("  %-20s %s\n", "watch-delay:", cfg.WatchDelay)
			pterm.Printf("  %-20s %d\n", "history-limit:", cfg.HistoryLimit)
			pterm.Println()

			pterm.DefaultSection.Printf("Categories (%d)\n", len(m.Categories))
			for i, cat := range m.Categories {
				pterm.Printf("  [%d] %s\n", i+1, pterm.Cyan(cat.Name))
				pterm.Printf("      %-22s %v\n", "enabled:", cat.IsEnabled())
				pterm.Printf("      %-22s %s\n", "source.path:", cat.Source.Path)
				pterm.Printf("      %-22s %s\n", "source.extensions:", strings.Join(cat.Source.Extensions, ", "))
				pterm.Printf("      %-22s %s\n", "destination.path:", cat.Destination.Path)
				pterm.Printf("      %-22s %s\n", "destination.conflict-strategy:", orDefault(cat.Destination.ConflictStrategy, "rename"))
				pterm.Printf("      %-22s %s\n", "destination.organize-by:", orDefault(cat.Destination.OrganizeBy, "(none)"))
				printFilterSummary(cat.Source.Filter)
				pterm.Println()
			}

			return nil
		},
	}
}

func printFilterSummary(f models.CategoryFilter) {
	if f.Regex != "" {
		pterm.Printf("      %-22s %s\n", "filter.regex:", f.Regex)
	}
	if f.Glob != "" {
		pterm.Printf("      %-22s %s\n", "filter.glob:", f.Glob)
	}
	if len(f.Ignore) > 0 {
		pterm.Printf("      %-22s %s\n", "filter.ignore:", strings.Join(f.Ignore, ", "))
	}
	if f.MinAge > 0 {
		pterm.Printf("      %-22s %s\n", "filter.min-age:", f.MinAge)
	}
	if f.MaxAge > 0 {
		pterm.Printf("      %-22s %s\n", "filter.max-age:", f.MaxAge)
	}
	if f.MinSize != "" {
		pterm.Printf("      %-22s %s\n", "filter.min-size:", f.MinSize)
	}
	if f.MaxSize != "" {
		pterm.Printf("      %-22s %s\n", "filter.max-size:", f.MaxSize)
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func orDefault(s, def string) string {
	if s == "" {
		return fmt.Sprintf("%s (default)", def)
	}
	return s
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
