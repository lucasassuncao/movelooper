// Package content detects a file's real media type from its magic bytes,
// independent of its filename or extension. It wraps gabriel-vasile/mimetype
// behind a small, config-agnostic API.
package content

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

// Info is the detected MIME information for a file.
type Info struct {
	Full string // full type without parameters, e.g. "image/png"
	Type string // top-level type, e.g. "image"
	Ext  string // canonical extension without the dot, e.g. "png" ("" if unknown)
}

// Detect reads the leading bytes of the file at path and returns its MIME info.
func Detect(path string) (Info, error) {
	m, err := mimetype.DetectFile(path)
	if err != nil {
		return Info{}, err
	}
	full := m.String()
	if i := strings.IndexByte(full, ';'); i >= 0 { // strip "; charset=utf-8"
		full = strings.TrimSpace(full[:i])
	}
	top, _, _ := strings.Cut(full, "/")
	return Info{
		Full: full,
		Type: top,
		Ext:  strings.TrimPrefix(m.Extension(), "."),
	}, nil
}
