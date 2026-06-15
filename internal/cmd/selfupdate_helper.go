package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lucasassuncao/movelooper/internal/updater"
)

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
		tags := make([]string, 0, 3)
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
		return strings.Clone(v[1:])
	}
	return v
}

func joinUpdateTags(s []string) string {
	return strings.Join(s, ", ")
}
