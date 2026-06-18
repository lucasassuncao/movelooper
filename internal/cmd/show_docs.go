package cmd

import (
	"fmt"
	"sort"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/docgenerator"
	"github.com/lucasassuncao/yedit/theme"

	"github.com/spf13/cobra"
)

func ShowCmd() *cobra.Command {
	var themeName string
	var listThemes bool

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

			return showDocs(t)
		},
	}

	cmd.Flags().StringVar(&themeName, "theme", "dark", "Theme name (run --list-themes to see options)")
	cmd.Flags().BoolVar(&listThemes, "list-themes", false, "List available theme names and exit")

	return cmd
}

func showDocs(t theme.Theme) error {
	zero := 0
	entries := []docgenerator.Entry{
		{Config: models.Configuration{}},
		{Config: models.Category{}, SplitStructs: true, RecursionLimit: &zero},
	}

	docs, err := docgenerator.GenerateInMemory(entries)
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	if err := docgenerator.RenderMarkdownDocsInTerminal(docs, "movelooper", t); err != nil {
		return fmt.Errorf("failed to render docs: %w", err)
	}
	return nil
}
