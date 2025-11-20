package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/helper"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// fileTracker keeps track of files detected by the watcher
type fileTracker struct {
	mu    sync.Mutex
	files map[string]time.Time // Path -> Time of last detection
}

// StartWatcher monitors directories and moves files in real-time until context is cancelled
func StartWatcher(ctx context.Context, m *models.Movelooper) error {
	// Refresh config
	m.Categories = config.UnmarshalConfig(m)

	stabilityThreshold := m.Viper.GetDuration("configuration.watch_delay")
	if stabilityThreshold == 0 {
		stabilityThreshold = 5 * time.Minute
	}

	m.Logger.Info("Starting Watch Mode", m.Logger.Args("stability_delay", stabilityThreshold.String()))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	tracker := &fileTracker{
		files: make(map[string]time.Time),
	}

	sources := make(map[string]bool)
	for _, cat := range m.Categories {
		if !sources[cat.Source] {
			m.Logger.Info("Monitoring directory", m.Logger.Args("path", cat.Source))
			if err := watcher.Add(cat.Source); err != nil {
				m.Logger.Error("Failed to watch directory", m.Logger.Args("path", cat.Source, "error", err.Error()))
			}
			sources[cat.Source] = true
		}
	}

	m.Logger.Info("Performing initial scan for existing files...")
	performInitialScan(m, tracker)

	// Ticker Loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Event Loop
	for {
		select {
		case <-ctx.Done():
			m.Logger.Info("Stopping Watch Mode")
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				tracker.mu.Lock()
				tracker.files[event.Name] = time.Now()
				tracker.mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			m.Logger.Error("Watcher error", m.Logger.Args("error", err.Error()))
		case <-ticker.C:
			processPendingFiles(m, tracker, stabilityThreshold)
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

			// Verifies if the extension is relevant for this category before tracking
			matchExtension := false
			fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))

			for _, ext := range cat.Extensions {
				if strings.ToLower(ext) == fileExt {
					matchExtension = true
					break
				}
			}

			if matchExtension {
				fullPath := filepath.Join(cat.Source, file.Name())
				tracker.files[fullPath] = time.Now()
			}
		}
	}
}

// processPendingFiles checks which files have "stabilized" (not used for the threshold duration) and attempts to move them
func processPendingFiles(m *models.Movelooper, tracker *fileTracker, threshold time.Duration) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()

	for path := range tracker.files {
		// Verify if the file still exists (it may have been deleted or moved manually)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			delete(tracker.files, path)
			continue
		}

		// Verifies if the file has stabilized based on its ModTime
		if now.Sub(info.ModTime()) > threshold {

			// Tries to find a category for this file
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			moved := false

			for _, cat := range m.Categories {
				// Verifies if the file is in the correct source folder for this category
				if filepath.Clean(filepath.Dir(path)) != filepath.Clean(cat.Source) {
					continue
				}

				for _, catExt := range cat.Extensions {
					if strings.EqualFold(catExt, ext) {
						files, _ := helper.ReadDirectory(cat.Source)
						var targetFile os.DirEntry
						for _, f := range files {
							if f.Name() == filepath.Base(path) {
								targetFile = f
								break
							}
						}

						if targetFile != nil {
							helper.MoveFiles(m, cat, []os.DirEntry{targetFile}, ext)
							moved = true
						}
						break
					}
				}
				if moved {
					break
				}
			}

			delete(tracker.files, path)
		}
	}
}
