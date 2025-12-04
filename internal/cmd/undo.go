package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// UndoCmd reverts the last batch of file moves
func UndoCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Undo the last file organization operation",
		Long:  `Reverts the most recent batch of moved files, moving them back to their source locations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.History == nil {
				return fmt.Errorf("history tracking is not initialized")
			}

			batchID, err := m.History.GetLastBatchID()
			if err != nil {
				return fmt.Errorf("failed to get last operation: %v", err)
			}

			entries := m.History.GetBatch(batchID)
			if len(entries) == 0 {
				m.Logger.Info("No moves found in the last batch.")
				return nil
			}

			// Build file list for confirmation prompt
			var fileList string
			for i, entry := range entries {
				fileName := filepath.Base(entry.Source)
				if i < 5 {
					fileList += fmt.Sprintf("  • %s\n", fileName)
				} else if i == 5 {
					fileList += fmt.Sprintf("  ... and %d more files\n", len(entries)-5)
					break
				}
			}

			confirmMessage := fmt.Sprintf("Undo operation for batch: %s\n\nFiles to restore (%d total):\n%s\nProceed with restore?",
				batchID, len(entries), fileList)

			var confirm bool
			err = huh.NewConfirm().
				Title(confirmMessage).
				Value(&confirm).
				Run()

			if err == huh.ErrUserAborted {
				m.Logger.Info("Undo operation cancelled by user")
				return nil
			}

			if !confirm {
				m.Logger.Info("Undo operation cancelled")
				return nil
			}

			// Proceed with undo
			m.Logger.Info("Undoing last operation...", m.Logger.Args("batch_id", batchID, "files", len(entries)))

			successCount := 0
			failCount := 0

			// Iterate in reverse order to handle potential dependencies
			for i := len(entries) - 1; i >= 0; i-- {
				entry := entries[i]

				// Check if destination file exists
				if _, err := os.Stat(entry.Destination); os.IsNotExist(err) {
					m.Logger.Warn("File not found at destination, skipping", m.Logger.Args("path", entry.Destination))
					failCount++
					continue
				}

				// Check if source location is clear
				if _, err := os.Stat(entry.Source); err == nil {
					m.Logger.Warn("Source location already occupied, skipping", m.Logger.Args("path", entry.Source))
					failCount++
					continue
				}

				// Ensure source directory exists
				sourceDir := filepath.Dir(entry.Source)
				if err := os.MkdirAll(sourceDir, 0755); err != nil {
					m.Logger.Error("Failed to create source directory", m.Logger.Args("path", sourceDir, "error", err.Error()))
					failCount++
					continue
				}

				// Move back
				if err := os.Rename(entry.Destination, entry.Source); err != nil {
					m.Logger.Error("Failed to move file back", m.Logger.Args("from", entry.Destination, "to", entry.Source, "error", err.Error()))
					failCount++
					continue
				}

				m.Logger.Info("Restored file", m.Logger.Args("path", entry.Source))
				successCount++
			}

			m.Logger.Info("Undo completed", m.Logger.Args("restored", successCount, "failed", failCount))

			// Remove batch from history
			if err := m.History.RemoveBatch(batchID); err != nil {
				m.Logger.Error("Failed to remove batch from history", m.Logger.Args("error", err.Error()))
			}

			return nil
		},
	}

	return cmd
}
