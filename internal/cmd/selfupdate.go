package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lucasassuncao/movelooper/internal/updater"
	"github.com/spf13/cobra"
)

// DefaultRepo is set at build time via ldflags.
var DefaultRepo = ""

// SelfUpdateCmd returns the self-update command.
func SelfUpdateCmd(currentVersion string) *cobra.Command {
	var (
		repo       string
		version    string
		list       bool
		prerelease bool
		limit      int
	)

	cmd := &cobra.Command{
		Use:               "self-update",
		Short:             "Update movelooper to a GitHub release",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Downloads a release of movelooper from GitHub and replaces the current binary.
The old binary is kept as movelooper.old until the next run.

With no flags, installs the latest stable release.
Use --list to see available versions, --version to pick a specific one,
and --prerelease to include rc/beta/alpha releases.`,
		Example: `  movelooper self-update
  movelooper self-update --list
  movelooper self-update --list --prerelease
  movelooper self-update --version v1.2.0
  movelooper self-update --repo lucasassuncao/movelooper`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return runSelfUpdateList(repo, prerelease, limit, currentVersion)
			}
			return updater.SelfUpdate(repo, "", currentVersion, version, prerelease)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", DefaultRepo, `GitHub repository in "owner/repo" format`)
	cmd.Flags().StringVar(&version, "version", "", "Install this specific release tag (e.g. v1.2.0) instead of the latest")
	cmd.Flags().BoolVar(&list, "list", false, "List available releases and exit")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include prereleases (rc/beta/alpha) in --list, or as the latest target when no --version is given")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of releases to show with --list (max 100)")

	return cmd
}

func runSelfUpdateList(repo string, includePrerelease bool, limit int, currentVersion string) error {
	if repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo lucasassuncao/movelooper)")
	}

	releases, err := updater.ListReleases(repo, "", includePrerelease, limit)
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		fmt.Printf("No releases found for %s.\n", repo)
		return nil
	}

	current := normalizeUpdateTag(currentVersion)
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, r := range releases {
		var tags []string
		if i == 0 && !r.Prerelease {
			tags = append(tags, "latest")
		}
		if r.Prerelease {
			tags = append(tags, "prerelease")
		}
		if normalizeUpdateTag(r.Tag) == current {
			tags = append(tags, "installed")
		}
		label := ""
		if len(tags) > 0 {
			label = "(" + joinUpdateTags(tags) + ")"
		}
		published := ""
		if !r.PublishedAt.IsZero() {
			published = r.PublishedAt.Format(time.DateOnly)
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", r.Tag, label, published)
	}
	return tw.Flush()
}

func normalizeUpdateTag(v string) string {
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		return v[1:]
	}
	return v
}

func joinUpdateTags(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
