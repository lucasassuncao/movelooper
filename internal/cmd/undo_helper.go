package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lucasassuncao/bezel/component/inline"
	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"golang.org/x/term"
)

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

// undoOptions carries the undo command's flags.
type undoOptions struct {
	DryRun, Force bool
	Yes           bool // restore without asking
	Categories    []string
}

// stdinIsTerminal is a variable so tests can run as if under cron.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) //#nosec G115 -- a stdin file descriptor always fits in an int
}

func undoBatch(ctx context.Context, m *models.Movelooper, batchID string, opts undoOptions) error {
	allEntries := m.History.GetBatch(batchID)
	if len(allEntries) == 0 {
		return fmt.Errorf("batch %q not found in history", batchID)
	}

	entries := allEntries
	if len(opts.Categories) > 0 {
		filtered, ok := filterEntriesByCategory(m, batchID, allEntries, opts.Categories)
		if !ok {
			return nil
		}
		entries = filtered
	}

	if opts.DryRun {
		return dryRunUndoBatch(m, batchID, entries, opts.Force)
	}

	if !opts.Yes {
		proceed, err := confirmUndo(m, batchID, entries)
		if err != nil || !proceed {
			return err
		}
	}

	restored := restoreEntries(ctx, m, entries, opts.Force)

	if len(restored) > 0 {
		if err := m.History.RemoveEntries(restored); err != nil {
			m.Logger.Error("failed to update history", m.Logger.Args("error", err.Error()))
		}
	}
	// Archive entries are never restored, so they do not count as failures.
	undoable := 0
	for _, e := range entries {
		if e.Action != string(models.ActionArchive) {
			undoable++
		}
	}
	if failed := undoable - len(restored); failed > 0 {
		return fmt.Errorf("%d file(s) could not be restored; they stay in history for another try", failed)
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
	filtered := make([]history.Entry, 0, len(all))
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
func dryRunUndoBatch(m *models.Movelooper, batchID string, entries []history.Entry, force bool) error {
	m.Logger.Info("[dry-run] would restore batch", m.Logger.Args("batch_id", batchID, "files", len(entries)))
	var restoreArgs, removeArgs []any
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Action == string(models.ActionArchive) {
			m.Logger.Warn("[dry-run] archive batches cannot be undone", m.Logger.Args("path", entry.Destination))
			continue
		}
		if _, err := os.Lstat(entry.Destination); os.IsNotExist(err) {
			m.Logger.Warn("[dry-run] file not found at destination, would skip", m.Logger.Args("path", entry.Destination))
			continue
		}
		if linkReplaced(entry) {
			m.Logger.Warn("[dry-run] would keep the file, it is no longer the link movelooper made", m.Logger.Args("path", entry.Destination))
			continue
		}
		switch entry.Action {
		case "copy", "symlink":
			// Undo removes the destination; the source still existing is expected.
			if risk := copyUndoRisk(entry); risk != "" && !force && entry.Action == string(models.ActionCopy) {
				m.Logger.Warn("[dry-run] would keep the copy, "+risk, m.Logger.Args("path", entry.Destination))
				continue
			}
			removeArgs = append(removeArgs, "path", entry.Destination)
		default:
			if _, err := os.Stat(entry.Source); err == nil {
				m.Logger.Warn("[dry-run] source location already occupied, would skip", m.Logger.Args("path", entry.Source))
				continue
			}
			restoreArgs = append(restoreArgs, "path", entry.Source)
		}
		if entry.Replaced {
			m.Logger.Warn("[dry-run] the file this one replaced at the destination was deleted by the conflict strategy and cannot come back",
				m.Logger.Args("path", entry.Destination))
		}
	}
	if len(removeArgs) > 0 {
		m.Logger.Info("[dry-run] would remove file(s)", m.Logger.Args(removeArgs...))
	}
	if len(restoreArgs) > 0 {
		m.Logger.Info("[dry-run] would restore file(s)", m.Logger.Args(restoreArgs...))
	}
	return nil
}

// confirmUndo asks before restoring and reports whether to go ahead. Without a
// terminal nobody can answer, so it fails instead of waiting on the prompt.
func confirmUndo(m *models.Movelooper, batchID string, entries []history.Entry) (bool, error) {
	if !stdinIsTerminal() {
		return false, errors.New("undo needs a terminal to confirm: pass --yes to restore without asking")
	}
	var sb strings.Builder
	shown := min(len(entries), 5)
	for _, entry := range entries[:shown] {
		fmt.Fprintf(&sb, "  - %s\n", filepath.Base(entry.Source))
	}
	if len(entries) > shown {
		fmt.Fprintf(&sb, "  ... and %d more files\n", len(entries)-shown)
	}
	msg := fmt.Sprintf("Undo batch: %s\n\nFiles to restore (%d total):\n%s\nProceed with restore?",
		batchID, len(entries), sb.String())

	confirm, err := inline.Confirm(msg, theme.Resolve(theme.Default, theme.DarkTerminal()))
	if err != nil || !confirm {
		m.Logger.Info("undo operation cancelled")
		return false, nil
	}
	return true, nil
}

// copyUndoRisk reports why removing the copy at the destination would destroy
// bytes that exist nowhere else, or "" when the removal is safe. Undoing a copy
// deletes the destination outright, so it is only safe while the original is
// still at the source and the copy has not been touched since it was made.
func copyUndoRisk(entry history.Entry) string {
	if _, err := os.Stat(entry.Source); os.IsNotExist(err) {
		return "the original is gone and this is the last copy of it"
	}
	// A copy carries the source's timestamps over (see fileops.copyFile), so a
	// destination newer than the run that created it was edited afterwards.
	if info, err := os.Stat(entry.Destination); err == nil && info.ModTime().After(entry.Timestamp) {
		return "it was modified after it was copied"
	}
	return ""
}

// restoreEntries moves files back to their source locations in reverse order.
// Returns the entries that were successfully restored so callers can remove
// only those from history, leaving failed restores available for retry.
// force removes a copy the guards in copyUndoRisk would otherwise preserve.
func restoreEntries(ctx context.Context, m *models.Movelooper, entries []history.Entry, force bool) []history.Entry {
	restored := make([]history.Entry, 0, len(entries))
	failCount := 0

	m.Logger.Info("undoing batch", m.Logger.Args("files", len(entries)))

	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		if entry.Action == string(models.ActionArchive) {
			m.Logger.Warn("archive batches cannot be undone; the archive file was left in place",
				m.Logger.Args("path", entry.Destination))
			continue
		}

		// Lstat: a link whose original is gone still holds the name.
		if _, err := os.Lstat(entry.Destination); os.IsNotExist(err) {
			m.Logger.Warn("file not found at destination, skipping", m.Logger.Args("path", entry.Destination))
			failCount++
			continue
		}
		if linkReplaced(entry) {
			m.Logger.Warn("keeping the file, it is no longer the link movelooper made", m.Logger.Args("path", entry.Destination))
			failCount++
			continue
		}

		// Undoing a copy deletes the destination, which is a real deletion and
		// not a reversal when the original is gone or the copy has since been
		// edited. Removing a symlink destroys no bytes, so it is not guarded.
		if entry.Action == string(models.ActionCopy) && !force {
			if risk := copyUndoRisk(entry); risk != "" {
				m.Logger.Warn("keeping the copy, "+risk+"; re-run with --force to remove it anyway",
					m.Logger.Args("path", entry.Destination))
				failCount++
				continue
			}
		}

		// The source checks only apply to move undo, which puts the file back at
		// the source. copy/symlink undo removes the destination, and the source
		// still existing is expected (those actions never consumed it).
		if entry.Action != string(models.ActionCopy) && entry.Action != string(models.ActionSymlink) {
			if _, err := os.Stat(entry.Source); err == nil {
				m.Logger.Warn("source location already occupied, skipping", m.Logger.Args("path", entry.Source))
				failCount++
				continue
			}

			sourceDir := filepath.Dir(entry.Source)
			if err := os.MkdirAll(sourceDir, 0o750); err != nil {
				m.Logger.Error("failed to create source directory", m.Logger.Args("path", sourceDir, "error", err.Error()))
				failCount++
				continue
			}
		}

		if err := restoreEntry(ctx, m, entry); err != nil {
			failCount++
			continue
		}
		if entry.Replaced {
			m.Logger.Warn("restored, but the file it replaced at the destination was deleted by the conflict strategy and cannot come back",
				m.Logger.Args("path", entry.Destination))
		}
		restored = append(restored, entry)
	}

	if len(restored) > 0 {
		restoredArgs := make([]any, 0, len(restored)*2)
		for _, entry := range restored {
			switch entry.Action {
			case string(models.ActionCopy), string(models.ActionSymlink):
				// Undo removed the destination; the source was never gone.
				restoredArgs = append(restoredArgs, "removed", entry.Destination)
			default:
				restoredArgs = append(restoredArgs, "path", entry.Source)
			}
		}
		m.Logger.Info("file(s) restored", m.Logger.Args(restoredArgs...))
	}

	m.Logger.Info("undo completed", m.Logger.Args("restored", len(restored), "failed", failCount))
	return restored
}

// linkReplaced reports whether a symlink entry's destination is no longer a
// link: something else took the name, and removing it would delete real bytes.
func linkReplaced(entry history.Entry) bool {
	if entry.Action != string(models.ActionSymlink) {
		return false
	}
	fi, err := os.Lstat(entry.Destination)
	return err == nil && fi.Mode()&os.ModeSymlink == 0
}

// restoreEntry performs the actual file operation for a single history entry.
func restoreEntry(ctx context.Context, m *models.Movelooper, entry history.Entry) error {
	switch entry.Action {
	case "copy", "symlink":
		if err := os.Remove(entry.Destination); err != nil {
			m.Logger.Error("failed to remove file", m.Logger.Args("path", entry.Destination, "error", err.Error()))
			return err
		}
	case history.ActionDedupe:
		// The destination holds the same bytes and stays: the duplicate comes back.
		if err := fileops.CopyFile(ctx, entry.Destination, entry.Source); err != nil {
			m.Logger.Error("failed to copy the duplicate back", m.Logger.Args("from", entry.Destination, "to", entry.Source, "error", err.Error()))
			return err
		}
	default: // "move" or legacy entries without Action
		if err := fileops.MoveFileCtx(ctx, entry.Destination, entry.Source); err != nil {
			m.Logger.Error("failed to move file back", m.Logger.Args("from", entry.Destination, "to", entry.Source, "error", err.Error()))
			return err
		}
	}
	return nil
}
