package cmd

import (
	"context"
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
	"github.com/lucasassuncao/movelooper/internal/fileops"
	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/scanner"
	"github.com/lucasassuncao/movelooper/internal/tokens"
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

// WatchOptions carries the CLI flags for the watch command.
type WatchOptions struct {
	DryRun          bool
	CategoryFilter  string
	IncludeDisabled bool
}

// watchConfig groups the runtime state shared by the ticker and pending-files loops.
type watchConfig struct {
	tracker   *fileTracker
	threshold time.Duration
	dryRun    bool
}

// WatchCmd defines the "watch" command to monitor directories and move files in real-time
func WatchCmd(m *models.Movelooper) *cobra.Command {
	var (
		dryRun          bool
		categoryFilter  string
		includeDisabled bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor folders and move files in real-time",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := WatchOptions{
				DryRun:          dryRun,
				CategoryFilter:  categoryFilter,
				IncludeDisabled: includeDisabled,
			}
			return runWatch(cmd.Context(), m, opts)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview mode - log matched files without moving them")
	cmd.Flags().StringVar(&categoryFilter, "category", "", "Comma-separated list of category names to monitor (default: all)")
	cmd.Flags().BoolVar(&includeDisabled, "include-disabled", false, "Include categories with enabled: false")
	return cmd
}

// runWatch sets up the file watcher and blocks until a shutdown signal is received.
func runWatch(ctx context.Context, m *models.Movelooper, opts WatchOptions) error {
	names := parseCategoryNames(opts.CategoryFilter)
	filtered, err := filterCategories(m.Categories, names, opts.IncludeDisabled, m.Logger)
	if err != nil {
		return err
	}
	m.Categories = filtered

	if opts.DryRun {
		m.Logger.Info("starting watch mode (dry-run)", m.Logger.Args("stability_delay", m.Config.WatchDelay.String()))
	} else {
		m.Logger.Info("starting watch mode", m.Logger.Args("stability_delay", m.Config.WatchDelay.String()))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	cfg := watchConfig{
		tracker:   &fileTracker{files: make(map[string]time.Time)},
		threshold: m.Config.WatchDelay,
		dryRun:    opts.DryRun,
	}

	registerSources(m, watcher)

	m.Logger.Info("performing initial scan for existing files")
	performInitialScan(m, cfg.tracker)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go runEventLoop(ctx, m, watcher, cfg.tracker)
	go runSignalHandler(m, cancel)
	go runTickerLoop(ctx, m, cfg)

	<-ctx.Done()
	return nil
}

// registerSources adds each unique source directory to the watcher.
func registerSources(m *models.Movelooper, watcher *fsnotify.Watcher) {
	seen := make(map[string]bool)
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		if seen[cat.Source.Path] {
			continue
		}
		m.Logger.Info("monitoring directory", m.Logger.Args("path", cat.Source.Path))
		if err := watcher.Add(cat.Source.Path); err != nil {
			m.Logger.Error("failed to watch directory", m.Logger.Args("path", cat.Source.Path, "error", err.Error()))
		}
		seen[cat.Source.Path] = true
	}
}

// runEventLoop captures fsnotify events and updates the tracker.
func runEventLoop(ctx context.Context, m *models.Movelooper, watcher *fsnotify.Watcher, tracker *fileTracker) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				alreadyTracked := func() bool {
					tracker.mu.Lock()
					defer tracker.mu.Unlock()
					_, tracked := tracker.files[event.Name]
					tracker.files[event.Name] = time.Now()
					return tracked
				}()
				if !alreadyTracked {
					m.Logger.Info("detected new file", m.Logger.Args("path", event.Name))
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			m.Logger.Error("watcher error", m.Logger.Args("error", err.Error()))
		case <-ctx.Done():
			return
		}
	}
}

// runSignalHandler calls cancel when SIGINT or SIGTERM is received.
func runSignalHandler(m *models.Movelooper, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	for range sigChan {
		m.Logger.Info("shutting down watch mode")
		cancel()
		return
	}
}

// runTickerLoop periodically checks for stable files and moves them.
func runTickerLoop(ctx context.Context, m *models.Movelooper, cfg watchConfig) {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			processPendingFiles(ctx, m, cfg)
		case <-ctx.Done():
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
		autoExclude := []string{cat.Destination.Path}
		entries, err := scanner.WalkSource(cat.Source, autoExclude)
		if err != nil {
			m.Logger.Warn("failed to scan directory during initial scan", m.Logger.Args("path", cat.Source.Path, "error", err.Error()))
			continue
		}

		for _, fe := range entries {
			if !filters.MatchesAnyExtension(fe.Entry.Name(), cat.Source.Extensions) {
				continue
			}
			info, err := fe.Entry.Info()
			if err != nil {
				continue
			}
			if !filters.MatchesFilter(cat.Source.Filter, fe.Entry.Name(), info) {
				continue
			}
			fullPath := filepath.Join(fe.Dir, fe.Entry.Name())
			// The Ticker will check the real ModTime of the file.
			// If the file is old, ModTime will be old and it will be moved on the first tick.
			tracker.files[fullPath] = time.Now()
		}
	}
}

// processPendingFiles checks which files have "stabilized" (not used for the threshold duration) and attempts to move them
func processPendingFiles(ctx context.Context, m *models.Movelooper, cfg watchConfig) {
	now := time.Now()

	// Snapshot tracked paths under lock to keep I/O outside the critical section
	paths := func() []string {
		cfg.tracker.mu.Lock()
		defer cfg.tracker.mu.Unlock()
		ps := make([]string, 0, len(cfg.tracker.files))
		for p := range cfg.tracker.files {
			ps = append(ps, p)
		}
		return ps
	}()

	for _, path := range paths {
		// Verify if the file still exists (it may have been deleted or moved manually)
		info, err := os.Stat(path)
		if err != nil {
			cfg.tracker.mu.Lock()
			delete(cfg.tracker.files, path)
			cfg.tracker.mu.Unlock()
			if !os.IsNotExist(err) {
				m.Logger.Warn("failed to stat tracked file, removing from tracker",
					m.Logger.Args("path", path, "error", err.Error()))
			}
			continue
		}

		// Verifies if the file has stabilized based on its ModTime
		if now.Sub(info.ModTime()) > cfg.threshold {
			if err := attemptMoveFile(ctx, m, path, cfg.dryRun); err != nil {
				m.Logger.Error("failed to move file", m.Logger.Args("path", path, "error", err.Error()))
			}
			// Remove from tracking after attempt (whether moved or ignored)
			cfg.tracker.mu.Lock()
			delete(cfg.tracker.files, path)
			cfg.tracker.mu.Unlock()
		}
	}
}

// resolveDryRunDest returns the destination directory that would be used for a
// given file and category, resolving the organize-by template when set.
func resolveDryRunDest(cat *models.Category, path string) string {
	destDir := cat.Destination.Path
	template := cat.Destination.OrganizeBy
	if template == "" {
		return destDir
	}
	info, err := os.Stat(path)
	if err != nil {
		return destDir
	}
	tctx := tokens.TokenContext{Info: info, CategoryName: cat.Name, Now: time.Now()}
	if subdir := tokens.ResolveGroupBy(template, tctx); subdir != "" {
		return filepath.Join(destDir, subdir)
	}
	return destDir
}

// attemptMoveFile tries to find a matching category and move the file.
// In dry-run mode it logs what would be moved without performing any I/O.
// Returns an error if a matching category was found but the move failed.
// Returns nil both when the file was moved successfully and when no category matched.
func attemptMoveFile(ctx context.Context, m *models.Movelooper, path string, dryRun bool) error {
	fileName := filepath.Base(path)
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		ext = filters.ExtAll
	}

	for _, cat := range m.Categories {
		if filepath.Clean(filepath.Dir(path)) != filepath.Clean(cat.Source.Path) {
			continue
		}
		if !matchesExtensionAndFilters(cat, fileName, path) {
			continue
		}
		if dryRun {
			m.Logger.Info("[dry-run] would move file",
				m.Logger.Args("file", fileName, "to", resolveDryRunDest(cat, path), "category", cat.Name))
			return nil
		}
		return moveFileToCategory(ctx, m, *cat, path, ext)
	}
	return nil
}

// matchesExtensionAndFilters reports whether the file matches the category's extension,
// name filters (regex/glob), and age/size constraints.
func matchesExtensionAndFilters(cat *models.Category, fileName, path string) bool {
	if !filters.MatchesAnyExtension(fileName, cat.Source.Extensions) {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return filters.MatchesFilter(cat.Source.Filter, fileName, info)
}

func moveFileToCategory(ctx context.Context, m *models.Movelooper, cat models.Category, path, ext string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file before move: %w", err)
	}

	targetFile := fileInfoDirEntry{info: info}
	batchID := history.NewWatchBatchID()
	moved := fileops.MoveFiles(ctx, fileops.MoveContext{Logger: m.Logger, History: m.History}, fileops.MoveRequest{
		Category:  &cat,
		Files:     []os.DirEntry{targetFile},
		Extension: ext,
		BatchID:   batchID,
		SourceDir: filepath.Dir(path),
	})
	if len(moved) == 0 {
		return fmt.Errorf("file was not moved: %s", filepath.Base(path))
	}
	return nil
}
