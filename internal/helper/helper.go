// Package helper provides utility functions for file and directory operations.
package helper

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
)

// MatchesRegex checks if the file name matches a pre-compiled regex pattern
func MatchesRegex(fileName string, re *regexp.Regexp) bool {
	return re.MatchString(fileName)
}

// MatchesIgnorePatterns reports whether fileName matches any of the provided
// glob patterns. Matching is case-insensitive. Patterns follow filepath.Match
// syntax: * matches any sequence of characters, ? matches one character.
func MatchesIgnorePatterns(fileName string, patterns []string) bool {
	lower := strings.ToLower(fileName)
	for _, pattern := range patterns {
		matched, err := filepath.Match(strings.ToLower(pattern), lower)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// MatchesGlob reports whether fileName matches the glob pattern.
// Supports brace expansion: *.{jpg,png} expands to *.jpg and *.png.
// Matching is case-insensitive.
func MatchesGlob(fileName, pattern string) bool {
	lower := strings.ToLower(fileName)
	for _, p := range expandGlobPattern(strings.ToLower(pattern)) {
		matched, err := filepath.Match(p, lower)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// expandGlobPattern expands a single {a,b,c} group into multiple patterns.
// For example, "*.{jpg,png}" becomes ["*.jpg", "*.png"].
// Only the first brace group is expanded; nested braces are not supported.
func expandGlobPattern(pattern string) []string {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")
	if start == -1 || end == -1 || end < start {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	alternatives := strings.Split(pattern[start+1:end], ",")

	expanded := make([]string, 0, len(alternatives))
	for _, alt := range alternatives {
		expanded = append(expanded, prefix+strings.TrimSpace(alt)+suffix)
	}
	return expanded
}

// ValidateGlob checks that pattern is syntactically valid after brace expansion.
func ValidateGlob(pattern string) error {
	for _, p := range expandGlobPattern(pattern) {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", p, err)
		}
	}
	return nil
}

// CreateDirectory checks if the specified directory exists, and if not, creates it with full permissions.
func CreateDirectory(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// ReadDirectory reads the contents of a given directory and returns the files.
func ReadDirectory(path string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// ValidateFiles checks each file in the provided list to see if it is a regular file
// and has the specified extension (case-insensitive). It returns the count of matching files.
func ValidateFiles(files []os.DirEntry, extension string) int {
	var count int

	for _, file := range files {
		if file.Type().IsRegular() && HasExtension(file, extension) {
			count++
		}
	}

	return count
}

// MoveFiles moves files with the specified extension from the source directory to the destination directory.
// The destination path includes a subdirectory named after the extension, avoiding overwriting files.
func MoveFiles(m *models.Movelooper, category *models.Category, files []os.DirEntry, extension, batchID string) {
	for _, file := range files {
		if !HasExtension(file, extension) {
			continue
		}

		sourcePath := filepath.Join(category.Source, file.Name())

		destDir := filepath.Join(category.Destination, extension)
		destPath := filepath.Join(destDir, file.Name())

		strategy := category.ConflictStrategy
		if strategy == "" {
			strategy = "rename"
		}
		resolved, skip := applyConflictStrategy(m, strategy, sourcePath, destPath, destDir, file.Name())
		if skip {
			continue
		}
		destPath = resolved
		err := moveFile(sourcePath, destPath)
		if err != nil {
			m.Logger.Error("failed to move file", m.Logger.Args("file", sourcePath), m.Logger.Args("error", err.Error()))
			continue
		}

		// Add to history if enabled
		if m.History != nil {
			err := m.History.Add(history.Entry{
				Source:      sourcePath,
				Destination: destPath,
				Timestamp:   time.Now(),
				BatchID:     batchID,
			})
			if err != nil {
				m.Logger.Warn("failed to add to history", m.Logger.Args("error", err.Error()))
			}
		}

		m.Logger.Info("successfully moved file", m.Logger.Args("source", sourcePath), m.Logger.Args("destination", destPath))
	}
}

// applyConflictStrategy checks whether destPath already exists and resolves the
// conflict according to strategy. It returns the final destination path and
// whether the file should be skipped entirely.
func applyConflictStrategy(m *models.Movelooper, strategy, sourcePath, destPath, destDir, fileName string) (resolved string, skip bool) {
	if _, err := os.Stat(destPath); err != nil {
		// Destination does not exist — no conflict.
		return destPath, false
	}
	resolvedPath, shouldMove, err := resolveConflict(strategy, sourcePath, destPath, destDir, fileName)
	if err != nil {
		m.Logger.Error("error to solve conflicts", m.Logger.Args("file", fileName), m.Logger.Args("error", err.Error()))
		return "", true
	}
	if !shouldMove {
		switch strategy {
		case "skip":
			m.Logger.Info("file skipped due to conflict strategy", m.Logger.Args("file", fileName))
		case "hash_check":
			m.Logger.Info("file identical, source removed", m.Logger.Args("file", fileName))
		}
		return "", true
	}
	return resolvedPath, false
}

// resolveConflict handles file name conflicts based on the specified strategy.
func resolveConflict(strategy, src, dst, destDir, fileName string) (string, bool, error) {
	switch strategy {
	case "overwrite":
		// Removes the destination file to allow overwrite
		if err := os.Remove(dst); err != nil {
			return "", false, fmt.Errorf("failed to remove destination file for overwrite: %w", err)
		}
		return dst, true, nil

	case "skip":
		return "", false, nil

	case "hash_check":
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
		// If contents are different but names are the same, fall through to default (rename)
		fallthrough

	case "rename":
		fallthrough
	default:
		return getUniqueDestinationPath(destDir, fileName), true, nil
	}
}

// compareFileHashes compares the SHA-256 hashes of two files to determine if they are identical
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

// calculateHash computes the SHA-256 hash of a file's contents
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

// moveFile attempts to move a file from source to destination.
// Falls back to copy+delete when os.Rename fails across different devices/drives.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// os.Rename fails across different filesystems/drives (EXDEV on Unix,
	// "The system cannot move the file to a different disk drive" on Windows).
	// Fall back to copy+delete.
	if !isCrossDeviceError(err) {
		return err
	}

	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("cross-device copy failed: %w", err)
	}

	if err := os.Remove(src); err != nil {
		// Copy succeeded but source cleanup failed — log-worthy but not fatal.
		// The file exists at destination, so we return nil to avoid a false failure.
		_ = err
	}

	return nil
}

// isCrossDeviceError reports whether err is a rename failure caused by src and
// dst being on different filesystems or drives. os.Rename always wraps such
// errors in *os.LinkError on both Windows and Unix.
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	return errors.As(err, &linkErr)
}

// copyFile copies src to dst, creating dst if needed.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// getUniqueDestinationPath ensures no file is overwritten by appending (n) if needed
func getUniqueDestinationPath(destDir, fileName string) string {
	ext := filepath.Ext(fileName)
	nameOnly := strings.TrimSuffix(fileName, ext)

	destPath := filepath.Join(destDir, fileName)
	counter := 1

	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		newName := fmt.Sprintf("%s(%d)%s", nameOnly, counter, ext)
		destPath = filepath.Join(destDir, newName)
		counter++
	}

	return destPath
}

// ParseSize parses a human-readable size string (e.g. "10MB", "1.5GB") into bytes.
// Supported suffixes (case-insensitive): B, KB, MB, GB, TB.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	suffixes := []struct {
		suffix     string
		multiplier int64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	upper := strings.ToUpper(s)
	for _, entry := range suffixes {
		if strings.HasSuffix(upper, entry.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(entry.suffix)])
			var val float64
			if _, err := fmt.Sscanf(numStr, "%f", &val); err != nil {
				return 0, fmt.Errorf("could not parse numeric value %q", numStr)
			}
			return int64(val * float64(entry.multiplier)), nil
		}
	}

	// No suffix — treat as raw bytes
	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, fmt.Errorf("unrecognised size format %q", s)
	}
	return val, nil
}

// MeetsMinAge reports whether the file's modification time is older than minAge.
// Always returns true when minAge is zero.
func MeetsMinAge(info os.FileInfo, minAge time.Duration) bool {
	if minAge == 0 {
		return true
	}
	return time.Since(info.ModTime()) >= minAge
}

// MeetsMinSize reports whether the file size is at least minSizeBytes.
// Always returns true when minSizeBytes is zero.
func MeetsMinSize(info os.FileInfo, minSizeBytes int64) bool {
	if minSizeBytes == 0 {
		return true
	}
	return info.Size() >= minSizeBytes
}

// MeetsMaxAge reports whether the file's modification time is newer than maxAge.
// Always returns true when maxAge is zero.
func MeetsMaxAge(info os.FileInfo, maxAge time.Duration) bool {
	if maxAge == 0 {
		return true
	}
	return time.Since(info.ModTime()) <= maxAge
}

// MeetsMaxSize reports whether the file size is at most maxSizeBytes.
// Always returns true when maxSizeBytes is zero.
func MeetsMaxSize(info os.FileInfo, maxSizeBytes int64) bool {
	if maxSizeBytes == 0 {
		return true
	}
	return info.Size() <= maxSizeBytes
}

// HasExtension checks if a file has a given extension (case-insensitive)
func HasExtension(file os.DirEntry, extension string) bool {
	ext := "." + extension
	fileExt := strings.ToLower(filepath.Ext(file.Name()))
	return fileExt == strings.ToLower(ext)
}

// GenerateLogArgs generates log arguments for a given extension.
func GenerateLogArgs(files []os.DirEntry, extension string) []interface{} {
	var logArgs []interface{}
	for _, file := range files {
		if HasExtension(file, extension) {
			logArgs = append(logArgs, "name", file.Name())
		}
	}
	return logArgs
}
