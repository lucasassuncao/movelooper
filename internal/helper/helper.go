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
	"runtime"
	"strings"
	"syscall"
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
			m.Logger.Error("failed to move file", m.Logger.Args("file", sourcePath, "error", err.Error()))
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

		m.Logger.Info("successfully moved file", m.Logger.Args("source", sourcePath, "destination", destPath))
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
		m.Logger.Error("error to solve conflicts", m.Logger.Args("file", fileName, "error", err.Error()))
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
	// ERROR_NOT_SAME_DEVICE on Windows). Fall back to copy+delete only for
	// that specific error — other errors (permissions, missing file) are returned as-is.
	if !isCrossDeviceError(err) {
		return err
	}

	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("cross-device copy failed: %w", err)
	}

	if err := os.Remove(src); err != nil {
		// Copy succeeded but source removal failed. Remove the destination copy
		// to avoid silent duplication, then surface the error.
		_ = os.Remove(dst)
		return fmt.Errorf("cross-device move: copied to %s but could not remove source: %w", dst, err)
	}

	return nil
}

// isCrossDeviceError reports whether err is a rename failure caused by src and
// dst being on different filesystems or drives.
//
// On Unix the kernel returns EXDEV; on Windows it returns ERROR_NOT_SAME_DEVICE
// (errno 17). Both are wrapped inside *os.LinkError by os.Rename, so we unwrap
// to the inner syscall error before comparing — this avoids treating unrelated
// *os.LinkError values (e.g. permission denied) as cross-device errors.
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}

	inner := linkErr.Err

	// syscall.EXDEV is defined on all Unix-like platforms.
	// On Windows, syscall.Errno(17) is ERROR_NOT_SAME_DEVICE.
	const windowsErrorNotSameDevice = syscall.Errno(17)

	switch runtime.GOOS {
	case "windows":
		return errors.Is(inner, windowsErrorNotSameDevice)
	default:
		return errors.Is(inner, syscall.EXDEV)
	}
}

// copyFile copies src to dst preserving the original file mode and timestamps.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}

	// Restore the original modification time so watchers and filters
	// that use file age (min-age / max-age) behave consistently.
	_ = os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())

	return nil
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

// MatchesAnyExtension reports whether fileName's extension matches any entry in the list.
// Comparison is case-insensitive; leading dots are stripped before comparing.
func MatchesAnyExtension(fileName string, extensions []string) bool {
	fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	for _, e := range extensions {
		if strings.ToLower(e) == fileExt {
			return true
		}
	}
	return false
}

// MatchesNameFilters reports whether fileName passes the category's regex and glob name
// filters. Returns true when neither filter is configured.
func MatchesNameFilters(fileName string, f models.CategoryFilter) bool {
	if f.CompiledRegex != nil && !MatchesRegex(fileName, f.CompiledRegex) {
		return false
	}
	if f.Glob != "" && !MatchesGlob(fileName, f.Glob) {
		return false
	}
	return true
}

// MeetsAgeSizeFilters reports whether info satisfies all age and size constraints
// defined in f. Returns true immediately when no constraints are set.
func MeetsAgeSizeFilters(info os.FileInfo, f models.CategoryFilter) bool {
	if f.MinAge == 0 && f.MaxAge == 0 && f.MinSizeBytes == 0 && f.MaxSizeBytes == 0 {
		return true
	}
	return MeetsMinAge(info, f.MinAge) &&
		MeetsMaxAge(info, f.MaxAge) &&
		MeetsMinSize(info, f.MinSizeBytes) &&
		MeetsMaxSize(info, f.MaxSizeBytes)
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
