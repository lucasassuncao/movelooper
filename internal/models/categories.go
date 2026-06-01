package models

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/logger"
)

// ParseCategoryNames splits a comma-separated category string into a slice of trimmed names.
// Returns nil when raw is empty or contains only separators.
func ParseCategoryNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var names []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			names = append(names, s)
		}
	}
	return names
}

// FilterCategories returns the subset of all that should be processed.
//
// When names is empty, all categories are returned. Without includeDisabled,
// categories with enabled: false are silently excluded (same behavior as today).
// With includeDisabled, all categories are returned regardless of their enabled field.
//
// When names is non-empty, each name is validated against the config. An unknown
// name returns an error. A disabled category without includeDisabled is skipped
// with a warning that suggests the flag.
func FilterCategories(all []*Category, names []string, includeDisabled bool, log logger.Logger) ([]*Category, error) {
	if len(names) == 0 {
		if includeDisabled {
			return all, nil
		}
		var result []*Category
		for _, cat := range all {
			if cat.IsEnabled() {
				result = append(result, cat)
			}
		}
		return result, nil
	}

	index := make(map[string]*Category, len(all))
	for _, cat := range all {
		index[cat.Name] = cat
	}

	var result []*Category
	for _, name := range names {
		cat, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("unknown category %q", name)
		}
		if !cat.IsEnabled() && !includeDisabled {
			log.Warn(fmt.Sprintf("category %q is disabled - use --include-disabled to run it anyway", name))
			continue
		}
		result = append(result, cat)
	}
	return result, nil
}
