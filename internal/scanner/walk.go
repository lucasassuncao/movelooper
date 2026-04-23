package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
)

// FileEntry pairs a regular file's containing directory with its DirEntry.
type FileEntry struct {
	Dir   string // absolute path of the directory containing Entry
	Entry os.DirEntry
}

// WalkSource returns all regular files under source.Path that pass the
// exclusion and depth rules. autoExclude lists destination paths that are
// automatically excluded to prevent infinite loops when the destination is
// inside the source tree. When source.Recursive is false only the top-level
// directory is read.
func WalkSource(source models.CategorySource, autoExclude []string) ([]FileEntry, error) {
	if source.Recursive && source.MaxDepth < 0 {
		return nil, fmt.Errorf("max-depth must be >= 0 (0 = unlimited), got %d", source.MaxDepth)
	}
	if !source.Recursive {
		return walkFlat(source.Path)
	}
	var results []FileEntry
	err := walkRecursive(source.Path, 0, source, autoExclude, &results)
	return results, err
}

// walkFlat reads a single directory and returns FileEntry for every regular file.
func walkFlat(dir string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []FileEntry
	for _, e := range entries {
		if e.Type().IsRegular() {
			result = append(result, FileEntry{Dir: dir, Entry: e})
		}
	}
	return result, nil
}

// walkRecursive descends into dir, collecting regular files while honouring
// exclusion rules and max-depth.
func walkRecursive(
	dir string,
	depth int,
	source models.CategorySource,
	autoExclude []string,
	results *[]FileEntry,
) error {
	if isExcluded(dir, autoExclude) || isExcluded(dir, source.ExcludePaths) {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Type().IsRegular() {
			*results = append(*results, FileEntry{Dir: dir, Entry: e})
			continue
		}
		if !e.IsDir() {
			continue // skip symlinks and other special files
		}
		childDepth := depth + 1
		if source.MaxDepth > 0 && childDepth > source.MaxDepth {
			continue // depth limit reached, do not descend
		}
		childDir := filepath.Join(dir, e.Name())
		if isExcluded(childDir, autoExclude) || isExcluded(childDir, source.ExcludePaths) {
			continue // skip before incurring the ReadDir syscall inside the recursive call
		}
		if err := walkRecursive(childDir, childDepth, source, autoExclude, results); err != nil {
			return err
		}
	}
	return nil
}

// isExcluded reports whether dir is equal to or a subdirectory of any path in list.
func isExcluded(dir string, list []string) bool {
	cleanDir := filepath.Clean(dir)
	for _, p := range list {
		cleanP := filepath.Clean(p)
		if cleanDir == cleanP {
			return true
		}
		rel, err := filepath.Rel(cleanP, cleanDir)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}
