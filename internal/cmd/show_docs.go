package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/docgenerator"
	"github.com/lucasassuncao/yedit/theme"

	"github.com/spf13/cobra"
)

func ShowCmd() *cobra.Command {
	var themeName string
	var listThemes bool
	var section string

	cmd := &cobra.Command{
		Use:               "show-docs",
		Short:             "Show documentation in terminal",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if listThemes {
				names := make([]string, 0, len(theme.All()))
				for name := range theme.All() {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					fmt.Println(name)
				}
				return nil
			}

			all := theme.All()
			t, ok := all[themeName]
			if !ok {
				return fmt.Errorf("unknown theme %q — run 'movelooper show-docs --list-themes' to see available themes", themeName)
			}

			return showDocs(t, section)
		},
	}

	cmd.Flags().StringVar(&themeName, "theme", "dark", "Theme name (run --list-themes to see options)")
	cmd.Flags().BoolVar(&listThemes, "list-themes", false, "List available theme names and exit")
	cmd.Flags().StringVar(&section, "section", "", "Show only the documentation for this topic (case-insensitive, partial match)")

	return cmd
}

func showDocs(t theme.Theme, section string) error {
	zero := 0
	entries := []docgenerator.Entry{
		{Config: models.Configuration{}},
		{Config: models.Category{}, SplitStructs: true, RecursionLimit: &zero},
	}

	docs, err := docgenerator.GenerateInMemory(entries)
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	if section != "" {
		filtered := filterDocSet(docs, section)
		if len(filtered.Pages) == 0 {
			available := make([]string, 0, len(docs.Pages))
			for name := range docs.Pages {
				available = append(available, strings.ToLower(name))
			}
			sort.Strings(available)
			return fmt.Errorf("no documentation found for section %q — available: %s", section, strings.Join(available, ", "))
		}
		docs = filtered
	}

	if err := docgenerator.RenderMarkdownDocsInTerminal(docs, "movelooper", t); err != nil {
		return fmt.Errorf("failed to render docs: %w", err)
	}
	return nil
}

// filterDocSet returns a new DocSet containing only pages whose name matches
// section (case-insensitive substring), plus any children of matched pages.
func filterDocSet(ds docgenerator.DocSet, section string) docgenerator.DocSet {
	q := strings.ToLower(section)
	out := docgenerator.DocSet{
		Pages:    make(map[string]string),
		Children: make(map[string][]string),
	}
	for name, page := range ds.Pages {
		if !strings.Contains(strings.ToLower(name), q) {
			continue
		}
		out.Pages[name] = page
		if children, ok := ds.Children[name]; ok {
			out.Children[name] = children
			for _, child := range children {
				if p, ok := ds.Pages[child]; ok {
					out.Pages[child] = p
				}
			}
		}
	}
	return out
}
