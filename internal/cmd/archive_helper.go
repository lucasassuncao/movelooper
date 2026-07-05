package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lucasassuncao/movelooper/internal/archive"
	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/pterm/pterm"
)

// archiveCategory packs all matched files of a category into one archive at the
// destination. It returns the final archive path (empty when nothing was
// written: no files, dry-run, or a conflict-strategy skip) and a non-nil error
// when the archive could not be written, so the caller can surface the failure
// through the run's exit code. Sources are deleted only when keep-source is
// false and the archive was written successfully.
func archiveCategory(ctx context.Context, m *models.Movelooper, category *models.Category, files []scanner.FileEntry, batch moveBatch) (string, error) {
	arc := category.Destination.Archive
	label := pterm.Cyan(fmt.Sprintf("[%s]", category.Name))
	if len(files) == 0 {
		m.Logger.Info(fmt.Sprintf("%s no files to archive", label))
		return "", nil
	}

	base := tokens.ResolveArchiveName(arc.Name, category.Name, time.Now())
	destPath := filepath.Join(category.Destination.Path, base+archive.Extension(archive.Format(arc.Format)))

	entries := archiveEntries(category, files)

	if batch.dryRun {
		args := make([]any, 0, len(entries)*2+2)
		args = append(args, "archive", destPath)
		for _, e := range entries {
			args = append(args, "entry", e.Name)
		}
		m.Logger.Info(fmt.Sprintf("%s would archive %d %s", label, len(entries), fileNoun("all", len(entries))), m.Logger.Args(args...))
		return "", nil
	}

	if err := fileops.CreateDirectory(category.Destination.Path); err != nil {
		return "", fmt.Errorf("create destination directory %q: %w", category.Destination.Path, err)
	}
	destPath, err := archiveConflictPath(m, category.Destination.ConflictStrategy, destPath)
	if err != nil {
		return "", err
	}
	if destPath == "" {
		return "", nil // skipped by conflict strategy
	}

	opts := archive.Options{
		Format:      archive.Format(arc.Format),
		Compression: archive.Compression(arc.Compression),
		OnProgress:  newArchiveProgress(m),
	}
	if err := archive.Write(ctx, destPath, entries, opts); err != nil {
		return "", fmt.Errorf("write archive %q: %w", destPath, err)
	}
	m.Logger.Info(fmt.Sprintf("%s archived %d %s", label, len(entries), fileNoun("all", len(entries))), m.Logger.Args("archive", destPath))

	recordArchiveHistory(m, category, destPath, batch)

	if !arc.KeepsSource() {
		deleteArchivedSources(m, files)
	}
	return destPath, nil
}

// archiveEntries builds (source, entry-name) pairs. With flatten=false the entry
// name preserves the file's path relative to the category source directory (so
// recursive scans keep their structure); otherwise the base name is used. Entry
// names are always slash-separated.
func archiveEntries(category *models.Category, files []scanner.FileEntry) []archive.Entry {
	flatten := category.Destination.Archive.Flatten
	root := category.Source.Path
	entries := make([]archive.Entry, 0, len(files))
	for _, fe := range files {
		src := filepath.Join(fe.Dir, fe.Entry.Name())
		name := fe.Entry.Name()
		if !flatten {
			if rel, err := filepath.Rel(root, src); err == nil && rel != "" && rel[0] != '.' {
				name = rel
			}
		}
		entries = append(entries, archive.Entry{Source: src, Name: filepath.ToSlash(name)})
	}
	return entries
}

// archiveConflictPath applies the conflict strategy to an already-existing
// archive path. Returns the path to write, or "" when the strategy says skip.
func archiveConflictPath(m *models.Movelooper, cs models.ConflictStrategy, destPath string) (string, error) {
	if _, err := os.Stat(destPath); err != nil {
		return destPath, nil // does not exist yet
	}
	switch cs {
	case models.ConflictStrategySkip:
		m.Logger.Info("archive already exists, skipping", m.Logger.Args("path", destPath))
		return "", nil
	case models.ConflictStrategyOverwrite:
		return destPath, nil
	default: // rename (and default)
		dir := filepath.Dir(destPath)
		name := filepath.Base(destPath)
		unique, err := fileops.UniqueDestination(dir, name)
		if err != nil {
			return "", fmt.Errorf("find a unique archive name for %q: %w", destPath, err)
		}
		return unique, nil
	}
}

func recordArchiveHistory(m *models.Movelooper, category *models.Category, destPath string, batch moveBatch) {
	if batch.recorder == nil {
		return
	}
	// One entry marks the archive batch; Action "archive" makes undo skip it.
	if err := batch.recorder.Add(history.Entry{
		Source:      category.Source.Path,
		Destination: destPath,
		Timestamp:   time.Now(),
		BatchID:     batch.batchID,
		Action:      string(models.ActionArchive),
		Category:    category.Name,
	}); err != nil {
		m.Logger.Warn("failed to record archive in history", m.Logger.Args("error", err.Error()))
	}
}

func deleteArchivedSources(m *models.Movelooper, files []scanner.FileEntry) {
	for _, fe := range files {
		p := filepath.Join(fe.Dir, fe.Entry.Name())
		if err := os.Remove(p); err != nil {
			m.Logger.Warn("failed to remove source after archiving", m.Logger.Args("path", p, "error", err.Error()))
		}
	}
}
