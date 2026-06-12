package cmd

import (
	"fmt"
	"log"

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
	entries := []docgenerator.Entry{
		{Config: models.Configuration{}},
		{Config: models.Category{}, SplitStructs: true},
	}

	docs, err := docgenerator.GenerateInMemory(entries)
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	if err := docgenerator.RenderMarkdownDocsInTerminal(docs, "movelooper"); err != nil {
		return fmt.Errorf("failed to render docs: %w", err)
	}
	return nil
}
