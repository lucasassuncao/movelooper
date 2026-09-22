package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/lucasassuncao/docgen"
	"github.com/lucasassuncao/movelooper/internal/models"

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

	// Reference pages are organised per config block, one directory each named
	// after the block's key in the config file. Configuration stays on one page;
	// Category splits, because its nested blocks are what people look up.
	zero, noSplit := 0, false
	entries := []docgen.Entry{
		{Config: models.Configuration{}, Slug: "configuration", SplitStructs: &noSplit},
		{Config: models.Category{}, Slug: "categories", RecursionLimit: &zero},
	}

	_, err := docgen.Generate(entries,
		docgen.WithMarkdown(attributesDir, docgen.Layout{
			Folders: docgen.FolderPerEntry,
			Files:   docgen.FilePerField,
		}),
		docgen.WithExamples(MovelooperBlockPresets, examplesDir, map[string]string{
			"configuration": "Configuration",
			"categories":    "Category",
		}),
		docgen.WithIndex(docsDir),
	)
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	// The JSON Schema describes the config file as a whole, so its root is
	// models.Config - the two blocks above are halves of one document, and a
	// language server needs the document.
	if _, err := docgen.Generate(
		[]docgen.Entry{{Config: models.Config{}}},
		docgen.WithJSONSchema(schemaDir),
	); err != nil {
		return fmt.Errorf("failed to generate json schema: %w", err)
	}

	fmt.Fprintf(w, "Documentation generated in '%s' directory.", docsDir)
	return nil
}
