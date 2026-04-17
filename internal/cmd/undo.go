package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// UndoCmd reverts a batch of file moves
func UndoCmd(m *models.Movelooper) *cobra.Command {
	var (
		listBatches    bool
		dryRun         bool
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
Use --category to undo only files from specific categories within a batch.`,
		Example: `  movelooper undo
  movelooper undo --list
  movelooper undo --dry-run
  movelooper undo batch_1718000000
  movelooper undo batch_1718000000 --dry-run
  movelooper undo --category images
  movelooper undo batch_1718000000 --category images,docs
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

			names := parseCategoryNames(categoryFilter)
			return undoBatch(m, batchID, dryRun, names)
		},
	}

	cmd.Flags().BoolVarP(&listBatches, "list", "l", false, "List all available batches")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what would be restored without moving any files")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to undo (default: all)")
	return cmd
}

func printBatchList(m *models.Movelooper) error {
	batches := m.History.GetAllBatches()
	if len(batches) == 0 {
		m.Logger.Info("no batches in history")
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

func undoBatch(m *models.Movelooper, batchID string, dryRun bool, categoryNames []string) error {
	allEntries := m.History.GetBatch(batchID)
	if len(allEntries) == 0 {
		return fmt.Errorf("batch %q not found in history", batchID)
	}

	partial := len(categoryNames) > 0
	entries := allEntries
	if partial {
		filtered, ok := filterEntriesByCategory(m, batchID, allEntries, categoryNames)
		if !ok {
			return nil
		}
		entries = filtered
	}

	if dryRun {
		return dryRunUndoBatch(m, batchID, entries)
	}

	if cancelled := confirmUndo(m, batchID, entries); cancelled {
		return nil
	}

	restoreEntries(m, entries)

	if partial {
		if err := m.History.RemoveCategoryFromBatch(batchID, categoryNames); err != nil {
			m.Logger.Error("failed to update history", m.Logger.Args("error", err.Error()))
		}
	} else {
		if err := m.History.RemoveBatch(batchID); err != nil {
			m.Logger.Error("failed to remove batch from history", m.Logger.Args("error", err.Error()))
		}
	}
	return nil
}

// filterEntriesByCategory returns the subset of entries matching the given categories.
// Returns (nil, false) when no matching entries are found (caller should return nil).
func filterEntriesByCategory(m *models.Movelooper, batchID string, all []history.Entry, categoryNames []string) ([]history.Entry, bool) {
	catSet := make(map[string]bool, len(categoryNames))
	for _, c := range categoryNames {
		catSet[c] = true
	}
	var filtered []history.Entry
	for _, e := range all {
		if e.Category == "" {
			m.Logger.Warn("skipping entry with unknown category (recorded before category tracking was added)",
				m.Logger.Args("source", e.Source))
			continue
		}
		if catSet[e.Category] {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		m.Logger.Warn("no entries for the specified categories in this batch",
			m.Logger.Args("batch_id", batchID, "categories", strings.Join(categoryNames, ",")))
		return nil, false
	}
	return filtered, true
}

// dryRunUndoBatch logs what would be restored without performing any file operations.
func dryRunUndoBatch(m *models.Movelooper, batchID string, entries []history.Entry) error {
	m.Logger.Info("[dry-run] would restore batch", m.Logger.Args("batch_id", batchID, "files", len(entries)))
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if _, err := os.Stat(entry.Destination); os.IsNotExist(err) {
			m.Logger.Warn("[dry-run] file not found at destination, would skip", m.Logger.Args("path", entry.Destination))
			continue
		}
		if _, err := os.Stat(entry.Source); err == nil {
			m.Logger.Warn("[dry-run] source location already occupied, would skip", m.Logger.Args("path", entry.Source))
			continue
		}
		switch entry.Action {
		case "copy", "symlink":
			m.Logger.Info("[dry-run] would remove file",
				m.Logger.Args("action", entry.Action, "path", entry.Destination))
		default:
			m.Logger.Info("[dry-run] would restore file",
				m.Logger.Args("from", entry.Destination, "to", entry.Source))
		}
	}
	return nil
}

// confirmUndo shows a confirmation prompt and returns true if the user cancelled.
func confirmUndo(m *models.Movelooper, batchID string, entries []history.Entry) bool {
	var fileList string
	for i, entry := range entries {
		if i < 5 {
			fileList += fmt.Sprintf("  - %s\n", filepath.Base(entry.Source))
		} else if i == 5 {
			fileList += fmt.Sprintf("  ... and %d more files\n", len(entries)-5)
			break
		}
	}
	msg := fmt.Sprintf("Undo batch: %s\n\nFiles to restore (%d total):\n%s\nProceed with restore?",
		batchID, len(entries), fileList)

	var confirm bool
	err := huh.NewConfirm().Title(msg).Value(&confirm).Run()
	if errors.Is(err, huh.ErrUserAborted) || !confirm {
		m.Logger.Info("undo operation cancelled")
		return true
	}
	return false
}

// restoreEntries moves files back to their source locations in reverse order.
func restoreEntries(m *models.Movelooper, entries []history.Entry) {
	successCount := 0
	failCount := 0

	m.Logger.Info("undoing batch", m.Logger.Args("files", len(entries)))

	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		if _, err := os.Stat(entry.Destination); os.IsNotExist(err) {
			m.Logger.Warn("file not found at destination, skipping", m.Logger.Args("path", entry.Destination))
			failCount++
			continue
		}
		if _, err := os.Stat(entry.Source); err == nil {
			m.Logger.Warn("source location already occupied, skipping", m.Logger.Args("path", entry.Source))
			failCount++
			continue
		}

		sourceDir := filepath.Dir(entry.Source)
		if err := os.MkdirAll(sourceDir, 0750); err != nil {
			m.Logger.Error("failed to create source directory", m.Logger.Args("path", sourceDir, "error", err.Error()))
			failCount++
			continue
		}

		if err := restoreEntry(m, entry); err != nil {
			failCount++
			continue
		}
		m.Logger.Info("file restored", m.Logger.Args("path", entry.Source))
		successCount++
	}

	m.Logger.Info("undo completed", m.Logger.Args("restored", successCount, "failed", failCount))
}

// restoreEntry performs the actual file operation for a single history entry.
func restoreEntry(m *models.Movelooper, entry history.Entry) error {
	switch entry.Action {
	case "copy", "symlink":
		if err := undoCopyOrSymlink(entry.Destination); err != nil {
			m.Logger.Error("failed to remove file", m.Logger.Args("path", entry.Destination, "error", err.Error()))
			return err
		}
	default: // "move" or legacy entries without Action
		if err := os.Rename(entry.Destination, entry.Source); err != nil {
			m.Logger.Error("failed to move file back", m.Logger.Args("from", entry.Destination, "to", entry.Source, "error", err.Error()))
			return err
		}
	}
	return nil
}

// undoCopyOrSymlink removes the destination file or symlink created by a copy or symlink action.
func undoCopyOrSymlink(dst string) error {
	return os.Remove(dst)
}
