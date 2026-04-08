package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// tickerInterval is how often the watch loop checks whether pending files have stabilized.
// Kept shorter than the default watch-delay so stable files are detected promptly.
const tickerInterval = 5 * time.Second

// fileInfoDirEntry adapts an os.FileInfo to the os.DirEntry interface.
// It is used in watch mode, where we obtain file metadata via os.Lstat
// rather than os.ReadDir, but downstream helpers expect an os.DirEntry.
type fileInfoDirEntry struct {
	info os.FileInfo
}

func (e fileInfoDirEntry) Name() string               { return e.info.Name() }
func (e fileInfoDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e fileInfoDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e fileInfoDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// fileTracker records files that the watcher has detected but not yet moved.
// Each entry maps an absolute file path to the time it was first seen in the
// current event burst. The ticker loop inspects these entries periodically and
// moves a file once its on-disk ModTime has been stable for longer than the
// configured watch-delay, indicating that the write is complete.
type fileTracker struct {
	mu    sync.Mutex
	files map[string]time.Time // absolute path → time of first detection
}

// WatchCmd defines the "watch" command to monitor directories and move files in real-time
func WatchCmd(m *models.Movelooper) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor folders and move files in real-time",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(m, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview mode — log matched files without moving them")
	return cmd
}

// runWatch sets up the file watcher and blocks until a shutdown signal is received.
func runWatch(m *models.Movelooper, dryRun bool) error {
	stabilityThreshold := m.Config.WatchDelay

	if dryRun {
		m.Logger.Info("starting watch mode (dry-run)", m.Logger.Args("stability_delay", stabilityThreshold.String()))
	} else {
		m.Logger.Info("starting watch mode", m.Logger.Args("stability_delay", stabilityThreshold.String()))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	tracker := &fileTracker{files: make(map[string]time.Time)}

	registerSources(m, watcher)

	m.Logger.Info("performing initial scan for existing files")
	performInitialScan(m, tracker)

	done := make(chan struct{})

	go runEventLoop(m, watcher, tracker, done)
	go runSignalHandler(m, done)
	go runTickerLoop(m, tracker, stabilityThreshold, dryRun, done)

	<-done
	return nil
}

// registerSources adds each unique source directory to the watcher.
func registerSources(m *models.Movelooper, watcher *fsnotify.Watcher) {
	seen := make(map[string]bool)
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		if seen[cat.Source] {
			continue
		}
		m.Logger.Info("monitoring directory", m.Logger.Args("path", cat.Source))
		if err := watcher.Add(cat.Source); err != nil {
			m.Logger.Error("failed to watch directory", m.Logger.Args("path", cat.Source, "error", err.Error()))
		}
		seen[cat.Source] = true
	}
}

// runEventLoop captures fsnotify events and updates the tracker.
func runEventLoop(m *models.Movelooper, watcher *fsnotify.Watcher, tracker *fileTracker, done <-chan struct{}) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				tracker.mu.Lock()
				tracker.files[event.Name] = time.Now()
				tracker.mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			m.Logger.Error("watcher error", m.Logger.Args("error", err.Error()))
		case <-done:
			return
		}
	}
}

// runSignalHandler closes done when SIGINT or SIGTERM is received.
func runSignalHandler(m *models.Movelooper, done chan struct{}) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	m.Logger.Info("shutting down watch mode")
	close(done)
}

// runTickerLoop periodically checks for stable files and moves them.
func runTickerLoop(m *models.Movelooper, tracker *fileTracker, threshold time.Duration, dryRun bool, done <-chan struct{}) {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			processPendingFiles(m, tracker, threshold, dryRun)
		case <-done:
			return
		}
	}
}

// performInitialScan verifies existing files in source directories and adds them to the tracker
func performInitialScan(m *models.Movelooper, tracker *fileTracker) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		files, err := helper.ReadDirectory(cat.Source)
		if err != nil {
			m.Logger.Warn("failed to read directory during initial scan", m.Logger.Args("path", cat.Source, "error", err.Error()))
			continue
		}

		for _, file := range files {
			if !file.Type().IsRegular() {
				continue
			}
			if !helper.MatchesAnyExtension(file.Name(), cat.Extensions) {
				continue
			}
			if helper.MatchesIgnorePatterns(file.Name(), cat.Filter.Ignore) {
				continue
			}
			if !helper.MatchesNameFilters(file.Name(), cat.Filter) {
				continue
			}
			fullPath := filepath.Join(cat.Source, file.Name())
			// The Ticker will check the real ModTime of the file.
			// If the file is old, ModTime will be old and it will be moved on the first tick.
			tracker.files[fullPath] = time.Now()
		}
	}
}

// processPendingFiles checks which files have "stabilized" (not used for the threshold duration) and attempts to move them
func processPendingFiles(m *models.Movelooper, tracker *fileTracker, threshold time.Duration, dryRun bool) {
	now := time.Now()

	// Snapshot tracked paths under lock to keep I/O outside the critical section
	tracker.mu.Lock()
	paths := make([]string, 0, len(tracker.files))
	for p := range tracker.files {
		paths = append(paths, p)
	}
	tracker.mu.Unlock()

	for _, path := range paths {
		// Verify if the file still exists (it may have been deleted or moved manually)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			tracker.mu.Lock()
			delete(tracker.files, path)
			tracker.mu.Unlock()
			continue
		}

		// Verifies if the file has stabilized based on its ModTime
		if err == nil && now.Sub(info.ModTime()) > threshold {
			if err := attemptMoveFile(m, path, dryRun); err != nil {
				m.Logger.Error("failed to move file", m.Logger.Args("path", path, "error", err.Error()))
			}
			// Remove from tracking after attempt (whether moved or ignored)
			tracker.mu.Lock()
			delete(tracker.files, path)
			tracker.mu.Unlock()
		}
	}
}

// attemptMoveFile tries to find a matching category and move the file.
// In dry-run mode it logs what would be moved without performing any I/O.
// Returns an error if a matching category was found but the move failed.
// Returns nil both when the file was moved successfully and when no category matched.
func attemptMoveFile(m *models.Movelooper, path string, dryRun bool) error {
	fileName := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(path), ".")

	for _, cat := range m.Categories {
		if filepath.Clean(filepath.Dir(path)) != filepath.Clean(cat.Source) {
			continue
		}
		if helper.MatchesIgnorePatterns(fileName, cat.Filter.Ignore) {
			continue
		}
		if matchesExtensionAndFilters(cat, fileName, path) {
			if dryRun {
				destDir := cat.Destination
				if cat.GroupByExtension {
					destDir = filepath.Join(cat.Destination, ext)
				}
				m.Logger.Info("[dry-run] would move file",
					m.Logger.Args("file", fileName, "to", destDir, "category", cat.Name))
				return nil
			}
			return moveFileToCategory(m, *cat, path, ext)
		}
	}
	return nil
}

// matchesExtensionAndFilters reports whether the file matches the category's extension,
// name filters (regex/glob), and age/size constraints.
func matchesExtensionAndFilters(cat *models.Category, fileName, path string) bool {
	if !helper.MatchesAnyExtension(fileName, cat.Extensions) {
		return false
	}
	if !helper.MatchesNameFilters(fileName, cat.Filter) {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return helper.MeetsAgeSizeFilters(info, cat.Filter)
}

func moveFileToCategory(m *models.Movelooper, cat models.Category, path, ext string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file before move: %w", err)
	}

	targetFile := fileInfoDirEntry{info: info}
	batchID := history.NewWatchBatchID()
	helper.MoveFiles(helper.MoveContext{Logger: m.Logger, History: m.History}, &cat, []os.DirEntry{targetFile}, ext, batchID)
	return nil
}
