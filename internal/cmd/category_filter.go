package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
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

// FilterCategories returns the categories to process; it keeps enabled ones by
// default, includes disabled ones when requested, and errors on unknown names.
func FilterCategories(all []*models.Category, names []string, includeDisabled bool, log logger.Logger) ([]*models.Category, error) {
	if len(names) == 0 {
		if includeDisabled {
			return all, nil
		}
		var result []*models.Category
		for _, cat := range all {
			if cat.IsEnabled() {
				result = append(result, cat)
			}
		}
		return result, nil
	}

	index := make(map[string]*models.Category, len(all))
	for _, cat := range all {
		index[cat.Name] = cat
	}

	var result []*models.Category
	for _, name := range names {
		cat, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("unknown category %q (valid categories: %s)", name, strings.Join(categoryNames(all), ", "))
		}
		if !cat.IsEnabled() && !includeDisabled {
			log.Warn(fmt.Sprintf("category %q is disabled - use --include-disabled to run it anyway", name))
			continue
		}
		result = append(result, cat)
	}
	return result, nil
}

// categoryNames returns the names of all categories, in config order.
func categoryNames(categories []*models.Category) []string {
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Name
	}
	return names
}
