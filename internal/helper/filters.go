package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
)

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

// ExtAll is the sentinel value that matches files of any extension.
const ExtAll = "all"

// HasExtension checks if a file has a given extension (case-insensitive).
// When extension is "all", every file matches.
func HasExtension(file os.DirEntry, extension string) bool {
	if strings.ToLower(extension) == ExtAll {
		return true
	}
	ext := "." + extension
	fileExt := strings.ToLower(filepath.Ext(file.Name()))
	return fileExt == strings.ToLower(ext)
}

// MatchesAnyExtension reports whether fileName's extension matches any entry in the list.
// Comparison is case-insensitive; leading dots are stripped before comparing.
// When the list contains "all", every file matches.
func MatchesAnyExtension(fileName string, extensions []string) bool {
	for _, e := range extensions {
		if strings.ToLower(e) == ExtAll {
			return true
		}
	}
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
	if f.CompiledRegex != nil && !f.CompiledRegex.MatchString(fileName) {
		return false
	}
	if f.Glob != "" && !MatchesGlob(fileName, f.Glob) {
		return false
	}
	return true
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
