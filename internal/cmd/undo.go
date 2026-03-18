package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// UndoCmd reverts a batch of file moves
func UndoCmd(m *models.Movelooper) *cobra.Command {
	var listBatches bool

	cmd := &cobra.Command{
		Use:   "undo [batch_id]",
		Short: "Undo a file organization operation",
		Long: `Reverts a batch of moved files, moving them back to their source locations.

Without arguments, reverts the most recent batch.
Pass a batch ID to revert a specific batch.
Use --list to see all available batches.`,
		Example: `  movelooper undo
  movelooper undo --list
  movelooper undo batch_1718000000
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
				var err error
				batchID, err = m.History.GetLastBatchID()
				if err != nil {
					return fmt.Errorf("failed to get last operation: %v", err)
				}
			}

			return undoBatch(m, batchID)
		},
	}

	cmd.Flags().BoolVarP(&listBatches, "list", "l", false, "List all available batches")
	return cmd
}

func printBatchList(m *models.Movelooper) error {
	batches := m.History.GetAllBatches()
	if len(batches) == 0 {
		m.Logger.Info("No batches in history.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "BATCH ID\tFILES\tTIMESTAMP")
	fmt.Fprintln(w, "--------\t-----\t---------")
	for _, b := range batches {
		fmt.Fprintf(w, "%s\t%d\t%s\n", b.BatchID, b.Count, b.Timestamp.Format("2006-01-02 15:04:05"))
	}
	return w.Flush()
}

func undoBatch(m *models.Movelooper, batchID string) error {
	entries := m.History.GetBatch(batchID)
	if len(entries) == 0 {
		return fmt.Errorf("batch %q not found in history", batchID)
	}

	// Build file list for confirmation prompt
	var fileList string
	for i, entry := range entries {
		if i < 5 {
			fileList += fmt.Sprintf("  • %s\n", filepath.Base(entry.Source))
		} else if i == 5 {
			fileList += fmt.Sprintf("  ... and %d more files\n", len(entries)-5)
			break
		}
	}

	confirmMessage := fmt.Sprintf("Undo batch: %s\n\nFiles to restore (%d total):\n%s\nProceed with restore?",
		batchID, len(entries), fileList)

	var confirm bool
	err := huh.NewConfirm().
		Title(confirmMessage).
		Value(&confirm).
		Run()
	if err == huh.ErrUserAborted || !confirm {
		m.Logger.Info("Undo operation cancelled")
		return nil
	}

	m.Logger.Info("Undoing batch...", m.Logger.Args("batch_id", batchID, "files", len(entries)))

	successCount := 0
	failCount := 0

	// Iterate in reverse order to handle potential dependencies
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		if _, err := os.Stat(entry.Destination); os.IsNotExist(err) {
			m.Logger.Warn("File not found at destination, skipping", m.Logger.Args("path", entry.Destination))
			failCount++
			continue
		}

		if _, err := os.Stat(entry.Source); err == nil {
			m.Logger.Warn("Source location already occupied, skipping", m.Logger.Args("path", entry.Source))
			failCount++
			continue
		}

		sourceDir := filepath.Dir(entry.Source)
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			m.Logger.Error("Failed to create source directory", m.Logger.Args("path", sourceDir, "error", err.Error()))
			failCount++
			continue
		}

		if err := os.Rename(entry.Destination, entry.Source); err != nil {
			m.Logger.Error("Failed to move file back", m.Logger.Args("from", entry.Destination, "to", entry.Source, "error", err.Error()))
			failCount++
			continue
		}

		m.Logger.Info("Restored file", m.Logger.Args("path", entry.Source))
		successCount++
	}

	m.Logger.Info("Undo completed", m.Logger.Args("restored", successCount, "failed", failCount))

	if err := m.History.RemoveBatch(batchID); err != nil {
		m.Logger.Error("Failed to remove batch from history", m.Logger.Args("error", err.Error()))
	}

	return nil
}
