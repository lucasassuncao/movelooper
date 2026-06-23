package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
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
)

const watchLockFile = "movelooper.lock"

// acquireWatchLock creates an exclusive lock file in the OS temp directory.
// Returns a release function that removes the file on clean shutdown.
func acquireWatchLock() (func(), error) {
	return acquireLockAt(filepath.Join(os.TempDir(), watchLockFile))
}

// acquireLockAt creates an exclusive lock file at path, recording the current
// PID. If the file already exists, the recorded PID decides the outcome: when
// that process is no longer running (a stale lock left by a killed instance) the
// lock is reclaimed; when it is still alive the call fails so two watchers never
// run at once. The returned function removes the lock on clean shutdown.
func acquireLockAt(path string) (func(), error) {
	release, err := createLockFile(path)
	if err == nil {
		return release, nil
	}
	if !os.IsExist(err) {
		return nil, fmt.Errorf("could not create lock file %s: %w", path, err)
	}

	if pid, ok := readLockPID(path); ok && processAlive(pid) {
		return nil, fmt.Errorf(
			"another instance of movelooper watch appears to be running (pid %d)\n"+
				"lock file: %s\n"+
				"if no instance is running, delete the file manually and retry",
			pid, path,
		)
	}

	// Stale lock (dead or unreadable PID): reclaim it and try once more.
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("could not remove stale lock file %s: %w", path, err)
	}
	release, err = createLockFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not reclaim stale lock file %s: %w", path, err)
	}
	return release, nil
}

// createLockFile creates path exclusively and writes the current PID into it.
func createLockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //#nosec G304 -- fixed filename in OS temp dir
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Close()
	return func() { os.Remove(path) }, nil
}

// readLockPID reads the PID recorded in a lock file. ok is false when the file
// cannot be read or does not hold a valid positive PID.
func readLockPID(path string) (pid int, ok bool) {
	data, err := os.ReadFile(path) //#nosec G304 -- fixed filename in OS temp dir
	if err != nil {
		return 0, false
	}
	pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// processAlive reports whether a process with the given PID is currently running.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false // Windows: FindProcess fails when the process does not exist
	}
	if runtime.GOOS == "windows" {
		_ = proc.Release()
		return true // Windows: a successful FindProcess means the process exists
	}
	// Unix: FindProcess always succeeds; probe liveness with signal 0.
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

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
// configured watch.delay, indicating that the write is complete.
type fileTracker struct {
	mu    sync.Mutex
	files map[string]time.Time // absolute path → time of first detection
}

// watchConfig groups the runtime state shared by the ticker and pending-files loops.
type watchConfig struct {
	tracker   *fileTracker
	threshold time.Duration
	showFiles bool
}

// categoriesWithHooks returns the names of categories that define before/after
// hooks. Hooks run only in the one-shot move command, so watch mode warns about
// these at startup rather than silently ignoring them.
func categoriesWithHooks(cats []*models.Category) []string {
	var names []string
	for _, cat := range cats {
		if cat.Hooks != nil && (cat.Hooks.Before != nil || cat.Hooks.After != nil) {
			names = append(names, cat.Name)
		}
	}
	return names
}

// runWatch sets up the file watcher and blocks until a shutdown signal is received.
func runWatch(ctx context.Context, m *models.Movelooper, opts WatchOptions) error {
	names := ParseCategoryNames(opts.CategoryFilter)
	filtered, err := FilterCategories(m.Categories, names, opts.IncludeDisabled, m.Logger)
	if err != nil {
		return err
	}
	m.Categories = filtered

	release, err := acquireWatchLock()
	if err != nil {
		return err
	}
	defer release()

	m.Logger.Info("starting watch mode", m.Logger.Args("stability_delay", m.Config.Watch.Delay.String()))

	for _, name := range categoriesWithHooks(m.Categories) {
		m.Logger.Warn("hooks are ignored in watch mode; they run only on the one-shot 'movelooper' command",
			m.Logger.Args("category", name))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	cfg := watchConfig{
		tracker:   &fileTracker{files: make(map[string]time.Time)},
		threshold: m.Config.Watch.Delay,
		showFiles: opts.ShowFiles,
	}

	registerSources(m, watcher)

	m.Logger.Info("performing initial scan for existing files")
	performInitialScan(ctx, m, cfg.tracker)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go runEventLoop(ctx, m, watcher, cfg.tracker)
	go runTickerLoop(ctx, m, &cfg)

	m.Logger.Info("watching for changes — press Ctrl+C to stop")

	<-ctx.Done()
	m.Logger.Info("shutting down watch mode")
	return nil
}

// registerSources adds each unique source directory to the watcher.
func registerSources(m *models.Movelooper, watcher *fsnotify.Watcher) {
	seen := make(map[string]bool, len(m.Categories))
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

// runTickerLoop periodically checks for stable files and moves them.
func runTickerLoop(ctx context.Context, m *models.Movelooper, cfg *watchConfig) {
	ticker := time.NewTicker(m.Config.Watch.PollInterval)
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

// performInitialScan verifies existing files in source directories and adds them to the tracker.
func performInitialScan(ctx context.Context, m *models.Movelooper, tracker *fileTracker) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		autoExclude := []string{cat.Destination.Path}
		entries, err := scanner.WalkSource(ctx, cat.Source, autoExclude)
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
			tracker.files[fullPath] = time.Now()
		}
	}
}

// processPendingFiles checks which files have stabilized and attempts to move them.
func processPendingFiles(ctx context.Context, m *models.Movelooper, cfg *watchConfig) {
	now := time.Now()

	snapshot := func() map[string]time.Time {
		cfg.tracker.mu.Lock()
		defer cfg.tracker.mu.Unlock()
		snap := make(map[string]time.Time, len(cfg.tracker.files))
		for p, t := range cfg.tracker.files {
			snap[p] = t
		}
		return snap
	}()

	for path, detected := range snapshot {
		if _, err := os.Stat(path); err != nil {
			cfg.tracker.mu.Lock()
			delete(cfg.tracker.files, path)
			cfg.tracker.mu.Unlock()
			if !os.IsNotExist(err) {
				m.Logger.Warn("failed to stat tracked file, removing from tracker",
					m.Logger.Args("path", path, "error", err.Error()))
			}
			continue
		}

		if now.Sub(detected) > cfg.threshold {
			if err := attemptMoveFile(ctx, m, path, cfg.showFiles); err != nil {
				if !os.IsNotExist(err) {
					m.Logger.Error("failed to move file", m.Logger.Args("path", path, "error", err.Error()))
				}
			}
			cfg.tracker.mu.Lock()
			delete(cfg.tracker.files, path)
			cfg.tracker.mu.Unlock()
		}
	}
}

// resolveDestDir returns the destination directory that would be used for a
// given file and category, resolving the organize-by template when set.
func resolveDestDir(cat *models.Category, path string) string {
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
	if subdir := tokens.ResolveGroupBy(template, &tctx); subdir != "" {
		return filepath.Join(destDir, subdir)
	}
	return destDir
}

// attemptMoveFile tries to find a matching category and move the file.
func attemptMoveFile(ctx context.Context, m *models.Movelooper, path string, showFiles bool) error {
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
		if showFiles {
			m.Logger.Info("moving file",
				m.Logger.Args("file", fileName, "to", resolveDestDir(cat, path), "category", cat.Name))
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
	result := fileops.MoveFiles(ctx, fileops.MoveContext{Logger: m.Logger, History: m.History}, fileops.MoveRequest{
		Category:  &cat,
		Files:     []os.DirEntry{targetFile},
		Extension: ext,
		BatchID:   batchID,
		SourceDir: filepath.Dir(path),
	})
	if len(result.Moved) == 0 {
		return fmt.Errorf("file was not moved: %s", filepath.Base(path))
	}
	return nil
}
