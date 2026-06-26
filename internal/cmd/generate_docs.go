package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

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

	exampleFiles, err := docgenerator.GenerateExampleDocs(examplesDir, MovelooperBlockPresets, map[string]string{
		"configuration": "Configuration",
		"categories":    "Category",
	})
	if err != nil {
		return fmt.Errorf("failed to generate examples: %w", err)
	}

	examplePages := make(map[string]bool, len(exampleFiles))
	for _, f := range exampleFiles {
		examplePages[strings.ToLower(f.Name)] = true
	}

	zero := 0
	entries := []docgenerator.Entry{
		{Config: models.Configuration{}, DocsDir: filepath.Join(attributesDir, "configuration")},
		{Config: models.Category{}, DocsDir: filepath.Join(attributesDir, "categories"), SplitStructs: true, RecursionLimit: &zero},
	}

	if err := docgenerator.Generate(docsDir, entries, docgenerator.WithExamples("../../examples", examplePages)); err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	fmt.Fprintf(w, "Documentation generated in '%s' directory.", docsDir)
	return nil
}
