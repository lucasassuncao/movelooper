package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/docgenerator"

	"github.com/spf13/cobra"
)

var GenerateCmd = &cobra.Command{
	Use:               "generate-docs",
	Short:             "Generate documentation for movelooper",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	RunE:              runGenerate,
	Hidden:            true,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	return generateDocs(cmd.OutOrStdout())
}

func generateDocs(w io.Writer) error {
	fmt.Fprintln(w, "Generating documentation...")

	docsDir := "docs/movelooper"

	entries := []docgenerator.Entry{
		{Config: models.Configuration{}, DocsDir: filepath.Join(docsDir, "configuration")},
		{Config: models.Category{}, DocsDir: filepath.Join(docsDir, "categories"), SplitStructs: true},
	}

	if err := docgenerator.Generate(docsDir, entries); err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	fmt.Fprintf(w, "Documentation generated in '%s' directory.", docsDir)
	return nil
}
