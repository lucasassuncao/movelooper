package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
)

// printFilterSummary prints the non-empty filter fields of a category filter.
func printFilterSummary(f models.CategoryFilter) {
	if f.Regex != "" {
		pterm.Printf("      %-32s %s\n", "source.filter.regex:", f.Regex)
	}
	if f.Glob != "" {
		pterm.Printf("      %-32s %s\n", "source.filter.glob:", f.Glob)
	}
	if len(f.Ignore) > 0 {
		pterm.Printf("      %-32s %s\n", "source.filter.ignore:", strings.Join(f.Ignore, ", "))
	}
	if f.MinAge > 0 {
		pterm.Printf("      %-32s %s\n", "source.filter.min-age:", f.MinAge)
	}
	if f.MaxAge > 0 {
		pterm.Printf("      %-32s %s\n", "source.filter.max-age:", f.MaxAge)
	}
	if f.MinSize != "" {
		pterm.Printf("      %-32s %s\n", "source.filter.min-size:", f.MinSize)
	}
	if f.MaxSize != "" {
		pterm.Printf("      %-32s %s\n", "source.filter.max-size:", f.MaxSize)
	}
}

// orDash returns s, or "-" if s is empty.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// orDefault returns s, or def if s is empty.
func orDefault(s, def string) string {
	if s == "" {
		return fmt.Sprintf("%s (default)", def)
	}
	return s
}
