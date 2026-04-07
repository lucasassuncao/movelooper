package helper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
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
	return getUniqueDestinationPath(destDir, fileName), true, nil
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
	file, err := os.Open(filePath)
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
	return getUniqueDestinationPath(destDir, fileName), true, nil
}
