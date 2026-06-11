package docs

import (
	"fmt"
	"log"

	"github.com/lucasassuncao/movelooper/internal/hints"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/docgenerator"

	"github.com/spf13/cobra"
)

var GenerateCmd = &cobra.Command{
	Use:               "generate-docs",
	Short:             "Generate documentation for movelooper",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run:               runGenerate,
	Hidden:            true,
}

func runGenerate(cmd *cobra.Command, args []string) {
	fmt.Println("Generating documentation...")

	docsDir := "docs/markdown"

	src, err := hints.Build()
	if err != nil {
		log.Fatalf("Failed to build hints: %v", err)
	}

	gen := docgenerator.NewSchemaGenerator(docgenerator.WithMetadata(src))

	names, err := gen.GenerateAllDocs(models.Config{}, docsDir)
	if err != nil {
		log.Fatalf("Failed to generate docs: %v", err)
	}

	if err := docgenerator.GenerateIndex(docsDir, names); err != nil {
		log.Fatalf("Failed to generate index: %v", err)
	}

	fmt.Println("Documentation generated in 'docs/markdown' directory.")
}
