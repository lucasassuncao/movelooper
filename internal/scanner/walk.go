// Package scanner walks a category's source directory and returns the regular
// files eligible for moving, honoring recursion, depth limits, and path
// exclusions.
package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
)

// FileEntry pairs a regular file's containing directory with its DirEntry.
type FileEntry struct {
	Dir   string // absolute path of the directory containing Entry
	Entry os.DirEntry
}

// SkippedDir records a sub-directory the walk could not read. It is reported
// rather than returned as an error: one unreadable folder should cost the files
// inside it, not the whole category.
type SkippedDir struct {
	Path string
	Err  error
}

// WalkSource returns all regular files under source.Path that pass the
// exclusion and depth rules. autoExclude lists destination paths that are
// automatically excluded to prevent infinite loops when the destination is
// inside the source tree. When source.Recursive is false only the top-level
// directory is read.
//
// Only a failure to read source.Path itself is an error. Sub-directories that
// cannot be read are skipped and returned in skipped, so a single folder the
// process has no permission on does not abandon the files it could reach.
func WalkSource(ctx context.Context, source models.CategorySource, autoExclude []string) (files []FileEntry, skipped []SkippedDir, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if source.Recursive && source.MaxDepth < 0 {
		return nil, nil, fmt.Errorf("max-depth must be >= 0 (0 = unlimited), got %d", source.MaxDepth)
	}
	if !source.Recursive {
		entries, err := walkFlat(ctx, source.Path)
		return entries, nil, err
	}
	var results []FileEntry
	var skips []SkippedDir
	err = walkRecursive(ctx, source.Path, 0, source, autoExclude, &results, &skips)
	return results, skips, err
}

// walkFlat reads a single directory and returns FileEntry for every regular file.
func walkFlat(ctx context.Context, dir string) ([]FileEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	ctx context.Context,
	dir string,
	depth int,
	source models.CategorySource,
	autoExclude []string,
	results *[]FileEntry,
	skipped *[]SkippedDir,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
		if err := walkRecursive(ctx, childDir, childDepth, source, autoExclude, results, skipped); err != nil {
			// A cancelled context stops the whole walk; anything else is this one
			// directory's problem and the rest of the tree still gets scanned.
			if ctx.Err() != nil {
				return err
			}
			*skipped = append(*skipped, SkippedDir{Path: childDir, Err: err})
		}
	}
	return nil
}

// isExcluded reports whether dir is equal to or a subdirectory of any path in list.
func isExcluded(dir string, list []string) bool {
	cleanDir := normalizePath(dir)
	for _, p := range list {
		cleanP := normalizePath(p)
		if cleanDir == cleanP {
			return true
		}
		rel, err := filepath.Rel(cleanP, cleanDir)
		if err == nil && !escapesParent(rel) {
			return true
		}
	}
	return false
}

// escapesParent reports whether a relative path leaves the directory it was
// computed from. Testing only for a ".." prefix would also catch a directory
// genuinely named "..cache", which is inside the parent, not outside it.
func escapesParent(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// normalizePath cleans p and, on Windows, lowercases it, so paths are compared
// the way the filesystem treats them. An exclude-path written with different
// casing than the directory on disk must still exclude it there. Lowercasing
// covers filepath.Rel too, which compares its elements exactly on every
// platform. Everywhere else the comparison stays exact, since two names
// differing only in case are two different directories.
func normalizePath(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}
