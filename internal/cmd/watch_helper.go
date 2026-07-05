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

// acquireWatchLock creates an exclusive lock file for watch mode.
// Returns a release function that removes the file on clean shutdown.
func acquireWatchLock() (func(), error) {
	path := watchLockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("could not create lock directory for %s: %w", path, err)
	}
	return acquireLockAt(path)
}

// watchLockPath returns the lock file location. It lives under ~/.movelooper
// (per-user, like logs and history) rather than the OS temp dir, which is
// shared between users on Unix and would let one user's watcher block
// another's. The temp dir remains only as a fallback when the home directory
// cannot be resolved.
func watchLockPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), watchLockFile)
	}
	return filepath.Join(home, ".movelooper", watchLockFile)
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //#nosec G304 -- fixed filename under the user's home (or OS temp dir as fallback)
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
	data, err := os.ReadFile(path) //#nosec G304 -- fixed filename under the user's home (or OS temp dir as fallback)
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

// maxWatchMoveRetries caps how many times a failed move is requeued before
// giving up until a new filesystem event re-tracks the file.
const maxWatchMoveRetries = 3

// watchConfig groups the runtime state shared by the ticker and pending-files loops.
type watchConfig struct {
	tracker   *fileTracker
	threshold time.Duration
	showFiles bool
	// retries counts consecutive failed move attempts per path. Only touched by
	// the single ticker goroutine, so no locking is needed.
	retries map[string]int
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

	for _, cat := range m.Categories {
		if cat.Destination.Action == models.ActionArchive {
			m.Logger.Warn("action archive is not supported in watch mode; the category will be skipped",
				m.Logger.Args("category", cat.Name))
		}
		if cat.Source.Recursive {
			m.Logger.Warn("recursive is not supported in watch mode; only the top-level source directory is monitored",
				m.Logger.Args("category", cat.Name))
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	cfg := watchConfig{
		tracker:   newFileTracker(),
		threshold: m.Config.Watch.Delay,
		showFiles: opts.ShowFiles,
		retries:   make(map[string]int),
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
				if !tracker.touch(event.Name, time.Now()) {
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

// performInitialScan verifies existing files in source directories and adds
// them to the tracker. The scan is deliberately non-recursive: fsnotify only
// watches the top-level source directory and attemptMoveFile only matches
// files directly under it, so tracking files from subdirectories would queue
// entries that can never be moved.
func performInitialScan(ctx context.Context, m *models.Movelooper, tracker *fileTracker) {
	for _, cat := range m.Categories {
		if !cat.IsEnabled() {
			continue
		}
		src := cat.Source
		src.Recursive = false
		autoExclude := []string{cat.Destination.Path}
		entries, err := scanner.WalkSource(ctx, src, autoExclude)
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
			fullPath := filepath.Join(fe.Dir, fe.Entry.Name())
			if !filters.MatchesFilter(cat.Source.Filter, fullPath, info) {
				continue
			}
			tracker.touch(fullPath, time.Now())
		}
	}
}

// processPendingFiles moves the files whose stability delay has elapsed. due()
// pops them off the heap atomically, so a file that received an event after the
// tick fired stays queued (its timestamp moved forward) instead of moving early.
// A failed move is requeued for another stability cycle up to
// maxWatchMoveRetries times, so a transient failure (e.g. a file briefly locked
// by another process) does not leave the file behind until a new event arrives.
func processPendingFiles(ctx context.Context, m *models.Movelooper, cfg *watchConfig) {
	for _, path := range cfg.tracker.due(time.Now(), cfg.threshold) {
		if _, err := os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				m.Logger.Warn("failed to stat tracked file, skipping",
					m.Logger.Args("path", path, "error", err.Error()))
			}
			delete(cfg.retries, path)
			continue
		}

		err := attemptMoveFile(ctx, m, path, cfg.showFiles)
		if err == nil || os.IsNotExist(err) {
			delete(cfg.retries, path)
			continue
		}

		cfg.retries[path]++
		if cfg.retries[path] < maxWatchMoveRetries {
			m.Logger.Warn("failed to move file, will retry",
				m.Logger.Args("path", path, "attempt", cfg.retries[path], "error", err.Error()))
			cfg.tracker.touch(path, time.Now())
			continue
		}
		delete(cfg.retries, path)
		m.Logger.Error("failed to move file, giving up until a new event re-tracks it",
			m.Logger.Args("path", path, "attempts", maxWatchMoveRetries, "error", err.Error()))
	}
}

// resolveDestDir returns the destination directory that would be used for a
// given file and category, resolving the organize-by template when set. It
// shares fileops.ResolveDestDir with the real move so the logged destination
// matches where the file actually lands.
func resolveDestDir(cat *models.Category, path string) string {
	if cat.Destination.OrganizeBy == "" {
		return cat.Destination.Path
	}
	info, err := os.Stat(path)
	if err != nil {
		return cat.Destination.Path
	}
	tctx := tokens.TokenContext{Info: info, CategoryName: cat.Name, Now: time.Now(), SourcePath: path}
	return fileops.ResolveDestDir(cat, &tctx)
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
	if cat.Destination.Action == models.ActionArchive {
		return false // archive is a batch operation, not supported in watch mode
	}
	if !filters.MatchesAnyExtension(fileName, cat.Source.Extensions) {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return filters.MatchesFilter(cat.Source.Filter, path, info)
}

func moveFileToCategory(ctx context.Context, m *models.Movelooper, cat models.Category, path, ext string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file before move: %w", err)
	}

	targetFile := fileInfoDirEntry{info: info}
	batchID := history.NewWatchBatchID()
	// Watch moves one file at a time, so saving per Add is fine here; assign the
	// concrete *History only when tracking is enabled to avoid a typed-nil Recorder.
	mctx := fileops.MoveContext{Logger: m.Logger}
	if m.History != nil {
		mctx.History = m.History
	}
	result := fileops.MoveFiles(ctx, mctx, fileops.MoveRequest{
		Category:    &cat,
		Files:       []os.DirEntry{targetFile},
		Extension:   ext,
		BatchID:     batchID,
		SourceDir:   filepath.Dir(path),
		LogEachMove: true,
	})
	if len(result.Moved) == 0 {
		if result.Skipped > 0 {
			// Skipped by the conflict strategy — a deliberate outcome, already
			// logged by the resolver, not a failure to retry.
			return nil
		}
		return fmt.Errorf("file was not moved: %s", filepath.Base(path))
	}
	return nil
}
