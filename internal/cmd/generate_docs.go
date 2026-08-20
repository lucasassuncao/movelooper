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
	attributesDir := filepath.Join(docsDir, "attributes")
	examplesDir := filepath.Join(docsDir, "examples")
	schemaDir := filepath.Join(docsDir, "schema")

	// Reference pages are organised per config block, one directory each.
	zero := 0
	entries := []docgenerator.Entry{
		{Config: models.Configuration{}, MarkdownDir: filepath.Join(attributesDir, "configuration")},
		{Config: models.Category{}, MarkdownDir: filepath.Join(attributesDir, "categories"), SplitStructs: true, RecursionLimit: &zero},
	}

	_, err := docgenerator.Generate(entries,
		docgenerator.WithMarkdown(attributesDir),
		docgenerator.WithExamples(MovelooperBlockPresets, examplesDir, map[string]string{
			"configuration": "Configuration",
			"categories":    "Category",
		}),
		docgenerator.WithIndex(docsDir),
	)
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	// The JSON Schema describes the config file as a whole, so its root is
	// models.Config - the two blocks above are halves of one document, and a
	// language server needs the document.
	if _, err := docgenerator.Generate(
		[]docgenerator.Entry{{Config: models.Config{}}},
		docgenerator.WithJSONSchema(schemaDir),
	); err != nil {
		return fmt.Errorf("failed to generate json schema: %w", err)
	}

	fmt.Fprintf(w, "Documentation generated in '%s' directory.", docsDir)
	return nil
}
