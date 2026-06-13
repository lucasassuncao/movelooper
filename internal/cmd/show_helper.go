package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
)

// printFilterSummary prints the non-empty filter fields of a category filter.
func printFilterSummary(f models.CategoryFilter) {
	if f.Match != nil {
		if f.Match.Glob != "" {
			pterm.Printf("      %-32s %s\n", "source.filter.match.glob:", f.Match.Glob)
		}
		if f.Match.Regex != "" {
			pterm.Printf("      %-32s %s\n", "source.filter.match.regex:", f.Match.Regex)
		}
		if f.Match.Literal != "" {
			pterm.Printf("      %-32s %s\n", "source.filter.match.literal:", f.Match.Literal)
		}
	}
	if f.Age != nil {
		if f.Age.Min > 0 {
			pterm.Printf("      %-32s %s\n", "source.filter.age.min:", f.Age.Min)
		}
		if f.Age.Max > 0 {
			pterm.Printf("      %-32s %s\n", "source.filter.age.max:", f.Age.Max)
		}
	}
	if f.Size != nil {
		if f.Size.Min != "" {
			pterm.Printf("      %-32s %s\n", "source.filter.size.min:", f.Size.Min)
		}
		if f.Size.Max != "" {
			pterm.Printf("      %-32s %s\n", "source.filter.size.max:", f.Size.Max)
		}
	}
	if len(f.Not) > 0 {
		nots := make([]string, 0, len(f.Not))
		for _, n := range f.Not {
			if n.Match != nil && n.Match.Glob != "" {
				nots = append(nots, n.Match.Glob)
			}
		}
		if len(nots) > 0 {
			pterm.Printf("      %-32s %s\n", "source.filter.not:", strings.Join(nots, ", "))
		}
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
