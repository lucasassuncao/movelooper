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
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// fileInfoDirEntry wraps an os.FileInfo to satisfy the os.DirEntry interface.
type fileInfoDirEntry struct {
	info os.FileInfo
}

func (e fileInfoDirEntry) Name() string               { return e.info.Name() }
func (e fileInfoDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e fileInfoDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e fileInfoDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// fileTracker keeps track of files detected by the watcher
type fileTracker struct {
	mu    sync.Mutex
	files map[string]time.Time // Path -> Time of last detection
}

// WatchCmd defines the "watch" command to monitor directories and move files in real-time
func WatchCmd(m *models.Movelooper) *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Monitor folders and move files in real-time",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(m)
		},
	}
}

// runWatch sets up the file watcher and blocks until a shutdown signal is received.
func runWatch(m *models.Movelooper) error {
	categories, err := config.UnmarshalConfig(m)
	if err != nil {
		return err
	}
	m.Categories = categories

	stabilityThreshold := m.Viper.GetDuration("configuration.watch-delay")
	if stabilityThreshold == 0 {
		stabilityThreshold = 5 * time.Minute
	}

	m.Logger.Info("Starting Watch Mode", m.Logger.Args("stability_delay", stabilityThreshold.String()))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	tracker := &fileTracker{files: make(map[string]time.Time)}

	registerSources(m, watcher)

	m.Logger.Info("Performing initial scan for existing files...")
	performInitialScan(m, tracker)

	done := make(chan struct{})

	go runEventLoop(m, watcher, tracker, done)
	go runSignalHandler(m, done)
	go runTickerLoop(m, tracker, stabilityThreshold, done)

	<-done
	return nil
}

// registerSources adds each unique source directory to the watcher.
func registerSources(m *models.Movelooper, watcher *fsnotify.Watcher) {
	seen := make(map[string]bool)
	for _, cat := range m.Categories {
		if seen[cat.Source] {
			continue
		}
		m.Logger.Info("Monitoring directory", m.Logger.Args("path", cat.Source))
		if err := watcher.Add(cat.Source); err != nil {
			m.Logger.Error("Failed to watch directory", m.Logger.Args("path", cat.Source, "error", err.Error()))
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
			m.Logger.Error("Watcher error", m.Logger.Args("error", err.Error()))
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
	m.Logger.Info("Shutting down watch mode...")
	close(done)
}

// runTickerLoop periodically checks for stable files and moves them.
func runTickerLoop(m *models.Movelooper, tracker *fileTracker, threshold time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			processPendingFiles(m, tracker, threshold)
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
		files, err := helper.ReadDirectory(cat.Source)
		if err != nil {
			continue
		}

		for _, file := range files {
			// Ignore non-regular files (directories, symlinks, etc.)
			if !file.Type().IsRegular() {
				continue
			}

			// Extensions are mandatory; check extension first
			fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))
			matches := false
			for _, ext := range cat.Extensions {
				if strings.ToLower(ext) == fileExt {
					matches = true
					break
				}
			}
			// Apply optional regex/glob name filter
			if matches && cat.Filter.CompiledRegex != nil {
				matches = helper.MatchesRegex(file.Name(), cat.Filter.CompiledRegex)
			}
			if matches && cat.Filter.Glob != "" {
				matches = helper.MatchesGlob(file.Name(), cat.Filter.Glob)
			}

			if matches && !helper.MatchesIgnorePatterns(file.Name(), cat.Filter.Ignore) {
				fullPath := filepath.Join(cat.Source, file.Name())
				// The Ticker will check the real ModTime of the file.
				// If the file is old, ModTime will be old and it will be moved on the first tick.
				tracker.files[fullPath] = time.Now()
			}
		}
	}
}

// processPendingFiles checks which files have "stabilized" (not used for the threshold duration) and attempts to move them
func processPendingFiles(m *models.Movelooper, tracker *fileTracker, threshold time.Duration) {
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
			attemptMoveFile(m, path)
			// Remove from tracking after attempt (whether moved or ignored)
			tracker.mu.Lock()
			delete(tracker.files, path)
			tracker.mu.Unlock()
		}
	}
}

// attemptMoveFile tries to find a matching category and move the file
func attemptMoveFile(m *models.Movelooper, path string) bool {
	fileName := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(path), ".")

	for _, cat := range m.Categories {
		if filepath.Clean(filepath.Dir(path)) != filepath.Clean(cat.Source) {
			continue
		}
		if helper.MatchesIgnorePatterns(fileName, cat.Filter.Ignore) {
			continue
		}
		if matchesExtensionAndFilters(cat, fileName, ext, path) {
			moveFileToCategory(m, *cat, path, ext)
			return true
		}
	}
	return false
}

// matchesExtensionAndFilters reports whether the file matches the category's extension,
// name filters (regex/glob), and age/size constraints.
func matchesExtensionAndFilters(cat *models.Category, fileName, ext, path string) bool {
	extMatched := false
	for _, catExt := range cat.Extensions {
		if strings.EqualFold(catExt, ext) {
			extMatched = true
			break
		}
	}
	if !extMatched {
		return false
	}
	if cat.Filter.CompiledRegex != nil && !helper.MatchesRegex(fileName, cat.Filter.CompiledRegex) {
		return false
	}
	if cat.Filter.Glob != "" && !helper.MatchesGlob(fileName, cat.Filter.Glob) {
		return false
	}
	return meetsWatchAgeSizeFilters(cat.Filter, path)
}

// meetsWatchAgeSizeFilters checks age and size constraints for a file path.
func meetsWatchAgeSizeFilters(f models.CategoryFilter, path string) bool {
	if f.MinAge == 0 && f.MaxAge == 0 && f.MinSizeBytes == 0 && f.MaxSizeBytes == 0 {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return helper.MeetsMinAge(info, f.MinAge) &&
		helper.MeetsMaxAge(info, f.MaxAge) &&
		helper.MeetsMinSize(info, f.MinSizeBytes) &&
		helper.MeetsMaxSize(info, f.MaxSizeBytes)
}

func moveFileToCategory(m *models.Movelooper, cat models.Category, path, ext string) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}

	targetFile := fileInfoDirEntry{info: info}
	batchID := fmt.Sprintf("watch_%d", time.Now().UnixNano())
	helper.MoveFiles(m, &cat, []os.DirEntry{targetFile}, ext, batchID)
}
