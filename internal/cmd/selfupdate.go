package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/updater"
	"github.com/spf13/cobra"
)

// DefaultRepo is set at build time via ldflags.
var DefaultRepo = ""

// SelfUpdateCmd returns the self-update command
func SelfUpdateCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:               "self-update",
		Short:             "Update movelooper to the latest GitHub release",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Downloads the latest movelooper release from GitHub and replaces the current binary.
The old binary is kept as movelooper.exe.old until the next run.`,
		Example: `  movelooper self-update
  movelooper self-update --repo lucasassuncao/movelooper`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.SelfUpdate(repo, "")
		},
	}

	cmd.Flags().StringVar(&repo, "repo", DefaultRepo, `GitHub repository in "owner/repo" format`)

	return cmd
}
