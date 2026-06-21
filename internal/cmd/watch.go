package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// WatchOptions carries the CLI flags for the watch command.
type WatchOptions struct {
	DryRun          bool
	ShowFiles       bool
	CategoryFilter  string
	IncludeDisabled bool
}

// WatchCmd defines the "watch" command to monitor directories and move files in real-time
func WatchCmd(m *models.Movelooper) *cobra.Command {
	var (
		dryRun          bool
		showFiles       bool
		categoryFilter  string
		includeDisabled bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor folders and move files in real-time",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := WatchOptions{
				DryRun:          dryRun,
				ShowFiles:       showFiles,
				CategoryFilter:  categoryFilter,
				IncludeDisabled: includeDisabled,
			}
			return runWatch(cmd.Context(), m, opts)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview mode - log matched files without moving them")
	cmd.Flags().BoolVar(&showFiles, "show-files", false, "Log each file and its destination as it is moved")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to monitor (default: all)")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include categories with enabled: false")
	return cmd
}
