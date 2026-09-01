package cmd

import (
	"fmt"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// UndoCmd reverts a batch of file moves
func UndoCmd(m *models.Movelooper) *cobra.Command {
	var (
		listBatches    bool
		dryRun         bool
		force          bool
		categoryFilter string
	)

	cmd := &cobra.Command{
		Use:   "undo [batch_id]",
		Short: "Undo a file organization operation",
		Long: `Reverts a batch of moved files, moving them back to their source locations.

Without arguments, reverts the most recent batch.
Pass a batch ID to revert a specific batch.
Use --list to see all available batches.
Use --dry-run to preview what would be restored without moving any files.
Use --category to undo only files from specific categories within a batch.

Undoing a copy deletes the file at the destination. That deletion is held back
when the original is no longer at the source, or when the copy was modified
after it was made: in both cases undo would destroy bytes rather than reverse
the run. Use --force to remove those copies anyway.`,
		Example: `  movelooper undo
  movelooper undo --list
  movelooper undo --dry-run
  movelooper undo batch_1718000000
  movelooper undo batch_1718000000 --dry-run
  movelooper undo --category images
  movelooper undo batch_1718000000 --category images,docs
  movelooper undo batch_1718000000 --force
  movelooper undo watch_1718000000000000000`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.History == nil {
				return fmt.Errorf("history tracking is not initialized")
			}

			if listBatches {
				return printBatchList(m)
			}

			var batchID string
			if len(args) == 1 {
				batchID = args[0]
			} else {
				batches := m.History.GetAllBatches()
				if len(batches) == 0 {
					m.Logger.Info("no batches in history")
					return nil
				}
				selected, err := pickBatch(batches, m.History)
				if err != nil {
					return fmt.Errorf("batch picker: %v", err)
				}
				if selected == "" {
					m.Logger.Info("undo operation cancelled")
					return nil
				}
				batchID = selected
			}

			names := ParseCategoryNames(categoryFilter)
			return undoBatch(cmd.Context(), m, batchID, dryRun, force, names)
		},
	}

	cmd.Flags().BoolVarP(&listBatches, "list", "l", false, "List all available batches")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what would be restored without moving any files")
	cmd.Flags().BoolVar(&force, "force", false, "Remove a copied file at the destination even when the original is gone or the copy was modified")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to undo (default: all)")
	_ = cmd.RegisterFlagCompletionFunc("category", categoryNameCompletion)
	return cmd
}
