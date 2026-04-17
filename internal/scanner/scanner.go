// Package scanner analyzes a directory and groups files by a built-in
// extension dictionary, producing a list of detected categories for use
// by the init command's --scan flag.
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectedCategory is a category found during a scan.
type DetectedCategory struct {
	Name       string   // category name from the dictionary
	Extensions []string // extensions actually found (no leading dot, e.g. "jpg")
}

// Result holds the outcome of a Scan call.
type Result struct {
	// Categories detected in the scanned path, in dictionary order.
	// Only categories with at least one matching file are included.
	Categories []DetectedCategory
}

// dictEntry is one entry in the built-in extension dictionary.
type dictEntry struct {
	name       string
	extensions []string // no leading dot
}

// dictionary is the ordered list of known categories.
// Order matters: it determines the order of categories in the generated config.
var dictionary = []dictEntry{
	{"images", []string{"jpg", "jpeg", "png", "gif", "webp", "heic", "heif", "bmp", "tiff", "svg"}},
	{"videos", []string{"mp4", "mkv", "mov", "avi", "wmv", "m4v", "webm", "flv"}},
	{"audio", []string{"mp3", "flac", "wav", "aac", "ogg", "m4a", "opus", "wma"}},
	{"documents", []string{"txt", "pdf", "docx", "doc", "xlsx", "xls", "pptx", "ppt", "odt", "ods", "odp"}},
	{"ebooks", []string{"epub", "mobi", "azw3"}},
	{"archives", []string{"zip", "tar", "gz", "rar", "7z", "bz2", "xz"}},
	{"fonts", []string{"ttf", "otf", "woff", "woff2"}},
	{"installers", []string{"exe", "msi", "apk", "pkg"}},
}

// Scan reads the top-level entries of path and returns the categories
// whose extensions appear at least once. Directories inside path are skipped.
// Returns an error if path does not exist or is not a directory.
func Scan(path string) (Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("path does not exist: %s", path)
		}
		return Result{}, fmt.Errorf("cannot stat path: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("path is not a directory: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return Result{}, fmt.Errorf("cannot read directory: %w", err)
	}

	// Build a set of extensions present in the directory (lowercase, no dot).
	present := make(map[string]struct{})
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(e.Name()), "."))
		if ext != "" {
			present[ext] = struct{}{}
		}
	}

	var detected []DetectedCategory
	for _, entry := range dictionary {
		var found []string
		for _, ext := range entry.extensions {
			if _, ok := present[ext]; ok {
				found = append(found, ext)
			}
		}
		if len(found) > 0 {
			detected = append(detected, DetectedCategory{
				Name:       entry.name,
				Extensions: found,
			})
		}
	}

	return Result{Categories: detected}, nil
}
