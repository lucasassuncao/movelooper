package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// showCmd returns the "config show" command that prints the active configuration with all defaults resolved
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
				pterm.Printf("      %-32s %v\n", "enabled:", cat.IsEnabled())
				pterm.Printf("      %-32s %s\n", "source.path:", cat.Source.Path)
				pterm.Printf("      %-32s %s\n", "source.extensions:", strings.Join(cat.Source.Extensions, ", "))
				pterm.Printf("      %-32s %v\n", "source.recursive:", cat.Source.Recursive)
				if cat.Source.Recursive {
					pterm.Printf("      %-32s %s\n", "source.max-depth:", orDefault(fmt.Sprintf("%d", cat.Source.MaxDepth), "0 (unlimited)"))
				}
				if len(cat.Source.ExcludePaths) > 0 {
					pterm.Printf("      %-32s %s\n", "source.exclude-paths:", strings.Join(cat.Source.ExcludePaths, ", "))
				}
				printFilterSummary(cat.Source.Filter)
				pterm.Printf("      %-32s %s\n", "destination.path:", cat.Destination.Path)
				pterm.Printf("      %-32s %s\n", "destination.action:", orDefault(string(cat.Destination.Action), "move"))
				pterm.Printf("      %-32s %s\n", "destination.conflict-strategy:", orDefault(string(cat.Destination.ConflictStrategy), "rename"))
				pterm.Printf("      %-32s %s\n", "destination.organize-by:", orDefault(cat.Destination.OrganizeBy, "(none)"))
				if cat.Destination.Rename != "" {
					pterm.Printf("      %-32s %s\n", "destination.rename:", cat.Destination.Rename)
				}
				if cat.Hooks != nil {
					if cat.Hooks.Before != nil {
						pterm.Printf("      %-32s shell=%s on-failure=%s\n", "hooks.before:", cat.Hooks.Before.Shell, cat.Hooks.Before.OnFailure)
					}
					if cat.Hooks.After != nil {
						pterm.Printf("      %-32s shell=%s on-failure=%s\n", "hooks.after:", cat.Hooks.After.Shell, cat.Hooks.After.OnFailure)
					}
				}
				pterm.Println()
			}

			return nil
		},
	}
}
