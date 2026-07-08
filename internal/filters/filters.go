package filters

import (
	"fmt"
	"math"
	"os"
	gpath "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lucasassuncao/movelooper/internal/content"
	"github.com/lucasassuncao/movelooper/internal/models"
)

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

// MatchesGlob reports whether fileName matches the glob pattern.
// Supports brace expansion: *.{jpg,png} expands to *.jpg and *.png.
func MatchesGlob(fileName, pattern string, caseSensitive bool) bool {
	name := normalizeCase(fileName, caseSensitive)
	for _, p := range expandGlobPattern(normalizeCase(pattern, caseSensitive)) {
		matched, err := filepath.Match(p, name)
		if err == nil && matched {
			return true
		}
	}
	return false
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

// ParseSize parses a human-readable size string (e.g. "10MB", "1.5GB",
// "256MiB") into bytes. Suffixes follow their standard meaning, matching the
// convention used by yedit's editor validators: KB/MB/GB/TB are decimal
// (powers of 1000) and KiB/MiB/GiB/TiB are binary (powers of 1024).
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Ordered longest-suffix-first so "B" never matches before "MB" or "GiB".
	suffixes := []struct {
		suffix     string
		multiplier int64
	}{
		{"TIB", 1 << 40},
		{"GIB", 1 << 30},
		{"MIB", 1 << 20},
		{"KIB", 1 << 10},
		{"TB", 1_000_000_000_000},
		{"GB", 1_000_000_000},
		{"MB", 1_000_000},
		{"KB", 1_000},
		{"B", 1},
	}

	upper := strings.ToUpper(s)
	for _, entry := range suffixes {
		if strings.HasSuffix(upper, entry.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(entry.suffix)])
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse numeric value %q", numStr)
			}
			if val < 0 {
				return 0, fmt.Errorf("size must not be negative: %q", s)
			}
			bytes := val * float64(entry.multiplier)
			if bytes > float64(math.MaxInt64) {
				return 0, fmt.Errorf("size out of range: %q", s)
			}
			return int64(bytes), nil
		}
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unrecognised size format %q", s)
	}
	if val < 0 {
		return 0, fmt.Errorf("size must not be negative: %q", s)
	}
	return val, nil
}

// MeetsMinAge reports whether the file's modification time is older than minAge.
func MeetsMinAge(info os.FileInfo, minAge time.Duration) bool {
	if minAge == 0 {
		return true
	}
	return time.Since(info.ModTime()) >= minAge
}

// MeetsMinSize reports whether the file size is at least minSizeBytes.
func MeetsMinSize(info os.FileInfo, minSizeBytes int64) bool {
	if minSizeBytes == 0 {
		return true
	}
	return info.Size() >= minSizeBytes
}

// MeetsMaxAge reports whether the file's modification time is newer than maxAge.
func MeetsMaxAge(info os.FileInfo, maxAge time.Duration) bool {
	if maxAge == 0 {
		return true
	}
	return time.Since(info.ModTime()) <= maxAge
}

// MeetsMaxSize reports whether the file size is at most maxSizeBytes.
func MeetsMaxSize(info os.FileInfo, maxSizeBytes int64) bool {
	if maxSizeBytes == 0 {
		return true
	}
	return info.Size() <= maxSizeBytes
}

// MeetsAgeSizeFilters reports whether info satisfies all age and size constraints.
func MeetsAgeSizeFilters(info os.FileInfo, f models.CategoryFilter) bool {
	if f.Age != nil {
		if !MeetsMinAge(info, f.Age.Min) || !MeetsMaxAge(info, f.Age.Max) {
			return false
		}
	}
	if f.Size != nil {
		if !MeetsMinSize(info, f.Size.MinBytes) || !MeetsMaxSize(info, f.Size.MaxBytes) {
			return false
		}
	}
	return true
}

// MatchesNameFilters reports whether fileName passes the category's name filter.
func MatchesNameFilters(fileName string, f models.CategoryFilter) bool {
	if f.Match == nil {
		return true
	}
	return matchesName(f.Match, fileName)
}

// MatchesFilter reports whether the file at path (with metadata info) passes
// filter f. path is the file's full path; the base name is used for name
// filters and the full path for MIME detection.
func MatchesFilter(f models.CategoryFilter, path string, info os.FileInfo) bool {
	// not is a modifier that excludes files and may coexist with any/all at the
	// same level, so it must be evaluated before the any/all branches return.
	for _, n := range f.Not {
		if MatchesFilter(n, path, info) {
			return false
		}
	}
	if len(f.Any) > 0 {
		for _, child := range f.Any {
			if MatchesFilter(child, path, info) {
				return true
			}
		}
		return false
	}
	if len(f.All) > 0 {
		for _, child := range f.All {
			if !MatchesFilter(child, path, info) {
				return false
			}
		}
		return true
	}
	if !MatchesNameFilters(filepath.Base(path), f) {
		return false
	}
	if !MeetsAgeSizeFilters(info, f) {
		return false
	}
	return matchesMimeFilter(f, path)
}

// matchesMimeFilter reports whether the file at path matches f.Mime, a glob
// (path.Match, slash-separated) against the detected MIME type. Empty f.Mime
// always matches; a detection error means no match for a positive mime rule.
func matchesMimeFilter(f models.CategoryFilter, path string) bool {
	if f.Mime == "" {
		return true
	}
	detected, err := content.Detect(path)
	if err != nil {
		return false
	}
	matched, err := gpath.Match(f.Mime, detected.Full)
	return err == nil && matched
}

func matchesName(m *models.MatchFilter, fileName string) bool {
	if m.CompiledRegex != nil && !m.CompiledRegex.MatchString(fileName) {
		return false
	}
	if m.Glob != "" && !MatchesGlob(fileName, m.Glob, m.CaseSensitive) {
		return false
	}
	if m.Literal != "" {
		if normalizeCase(fileName, m.CaseSensitive) != normalizeCase(m.Literal, m.CaseSensitive) {
			return false
		}
	}
	return true
}

// GenerateLogArgs generates log arguments for a given extension.
func GenerateLogArgs(files []os.DirEntry, extension string) []interface{} {
	logArgs := make([]interface{}, 0, len(files)*2)
	for _, file := range files {
		if HasExtension(file, extension) {
			logArgs = append(logArgs, "name", file.Name())
		}
	}
	return logArgs
}

func normalizeCase(s string, caseSensitive bool) string {
	if caseSensitive {
		return s
	}
	return strings.ToLower(s)
}

// expandGlobPattern expands brace groups into a list of plain glob patterns.
// Groups may appear in sequence ("{a,b}/{c,d}" expands to the cartesian
// product), but nesting ("{a,{b,c}}") is not supported, and an unmatched brace
// leaves the pattern untouched (treated as a literal).
func expandGlobPattern(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start == -1 {
		return []string{pattern}
	}
	// Search for "}" only after start, so a literal "}" earlier in the pattern
	// (e.g. "a}b{c,d}") does not hide a valid group that follows it.
	end := strings.Index(pattern[start:], "}")
	if end == -1 {
		return []string{pattern}
	}
	end += start

	prefix := pattern[:start]
	alternatives := strings.Split(pattern[start+1:end], ",")
	// Recurse on the tail so multiple groups expand to their cartesian product,
	// e.g. "{a,b}/{c,d}" -> a/c, a/d, b/c, b/d.
	rest := expandGlobPattern(pattern[end+1:])

	expanded := make([]string, 0, len(alternatives)*len(rest))
	for _, alt := range alternatives {
		for _, r := range rest {
			expanded = append(expanded, prefix+strings.TrimSpace(alt)+r)
		}
	}
	return expanded
}
