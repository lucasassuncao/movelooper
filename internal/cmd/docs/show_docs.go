package docs

import (
	"fmt"
	"log"

	"github.com/lucasassuncao/movelooper/internal/hints"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/docgenerator"

	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:               "show-docs",
	Short:             "Show documentation in terminal",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run:               runShow,
}

func runShow(cmd *cobra.Command, args []string) {
	if err := showDocs(); err != nil {
		log.Fatalf("%v", err)
	}
}

func showDocs() error {
	src, err := hints.Build()
	if err != nil {
		return fmt.Errorf("failed to build hints: %w", err)
	}

	gen := docgenerator.NewSchemaGenerator(docgenerator.WithMetadata(src))
	docs := gen.GenerateDocsInMemory(models.Config{})

	if err := docgenerator.RenderMarkdownDocsInTerminal(docs, "movelooper"); err != nil {
		return fmt.Errorf("failed to render docs: %w", err)
	}
	return nil
}
