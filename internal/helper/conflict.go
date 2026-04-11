package helper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ConflictResolver resolves a naming conflict when a destination file already exists.
// Resolve returns the final destination path, whether the move should proceed, and
// any error encountered. When shouldMove is false the caller must skip the file.
type ConflictResolver interface {
	Resolve(src, dst, destDir, fileName string) (resolvedPath string, shouldMove bool, err error)
}

// conflictResolvers maps strategy names to their ConflictResolver implementation.
// Add new strategies here without modifying any existing code.
var conflictResolvers = map[string]ConflictResolver{
	"rename":     &renameResolver{},
	"overwrite":  &overwriteResolver{},
	"skip":       &skipResolver{},
	"hash_check": &hashCheckResolver{},
	"newest":     &newestResolver{},
	"oldest":     &oldestResolver{},
	"larger":     &largerResolver{},
	"smaller":    &smallerResolver{},
}

// resolveConflict dispatches to the registered ConflictResolver for strategy.
// Falls back to renameResolver for unknown or empty strategy names.
func resolveConflict(strategy, src, dst, destDir, fileName string) (string, bool, error) {
	resolver, ok := conflictResolvers[strategy]
	if !ok {
		resolver = conflictResolvers["rename"]
	}
	return resolver.Resolve(src, dst, destDir, fileName)
}

// renameResolver appends (n) to the file name until a free slot is found.
type renameResolver struct{}

func (r *renameResolver) Resolve(_, _, destDir, fileName string) (string, bool, error) {
	path, err := getUniqueDestinationPath(destDir, fileName)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

// overwriteResolver removes the existing destination file so the source can take its place.
type overwriteResolver struct{}

func (r *overwriteResolver) Resolve(_, dst, _, _ string) (string, bool, error) {
	if err := os.Remove(dst); err != nil {
		return "", false, fmt.Errorf("failed to remove destination file for overwrite: %w", err)
	}
	return dst, true, nil
}

// skipResolver signals that the file should not be moved.
type skipResolver struct{}

func (r *skipResolver) Resolve(_, dst, _, _ string) (string, bool, error) {
	return "", false, nil
}

// compareFileHashes reports whether file1 and file2 have identical SHA-256 digests.
func compareFileHashes(file1, file2 string) (bool, error) {
	h1, err := calculateHash(file1)
	if err != nil {
		return false, err
	}
	h2, err := calculateHash(file2)
	if err != nil {
		return false, err
	}
	return h1 == h2, nil
}

// calculateHash computes the SHA-256 hash of a file's contents.
func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath)) //#nosec G304 -- path comes from directory walk
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// newestResolver keeps whichever file has the most recent modification time.
// If the destination is newer or equal, the source is skipped.
// If the source is newer, the destination is overwritten.
type newestResolver struct{}

func (r *newestResolver) Resolve(src, dst, _, _ string) (string, bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return "", false, err
	}
	if !srcInfo.ModTime().After(dstInfo.ModTime()) {
		return "", false, nil // destination is newer or equal — keep it
	}
	if err := os.Remove(dst); err != nil {
		return "", false, fmt.Errorf("newest: failed to remove older destination: %w", err)
	}
	return dst, true, nil
}

// oldestResolver keeps whichever file has the oldest modification time.
// If the destination is older or equal, the source is skipped.
// If the source is older, the destination is overwritten.
type oldestResolver struct{}

func (r *oldestResolver) Resolve(src, dst, _, _ string) (string, bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return "", false, err
	}
	if !srcInfo.ModTime().Before(dstInfo.ModTime()) {
		return "", false, nil // destination is older or equal — keep it
	}
	if err := os.Remove(dst); err != nil {
		return "", false, fmt.Errorf("oldest: failed to remove newer destination: %w", err)
	}
	return dst, true, nil
}

// largerResolver keeps the larger of the two files.
// If the destination is larger or equal, the source is skipped.
// If the source is larger, the destination is overwritten.
type largerResolver struct{}

func (r *largerResolver) Resolve(src, dst, _, _ string) (string, bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return "", false, err
	}
	if srcInfo.Size() <= dstInfo.Size() {
		return "", false, nil // destination is larger or equal — keep it
	}
	if err := os.Remove(dst); err != nil {
		return "", false, fmt.Errorf("larger: failed to remove smaller destination: %w", err)
	}
	return dst, true, nil
}

// smallerResolver keeps the smaller of the two files.
// If the destination is smaller or equal, the source is skipped.
// If the source is smaller, the destination is overwritten.
type smallerResolver struct{}

func (r *smallerResolver) Resolve(src, dst, _, _ string) (string, bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return "", false, err
	}
	if srcInfo.Size() >= dstInfo.Size() {
		return "", false, nil // destination is smaller or equal — keep it
	}
	if err := os.Remove(dst); err != nil {
		return "", false, fmt.Errorf("smaller: failed to remove larger destination: %w", err)
	}
	return dst, true, nil
}

// hashCheckResolver compares source and destination by SHA-256 hash.
// If identical, the source is removed (deduplication). If different,
// it falls back to rename behaviour.
type hashCheckResolver struct{}

func (r *hashCheckResolver) Resolve(src, dst, destDir, fileName string) (string, bool, error) {
	match, err := compareFileHashes(src, dst)
	if err != nil {
		return "", false, err
	}
	if match {
		if err := os.Remove(src); err != nil {
			return "", false, fmt.Errorf("failed to remove duplicate source file: %w", err)
		}
		return "", false, nil
	}
	// Files differ — rename to avoid clobbering the existing destination.
	path, err := getUniqueDestinationPath(destDir, fileName)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}
