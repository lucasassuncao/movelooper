package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/lucasassuncao/yedit/spec"
	"gopkg.in/yaml.v3"
)

// configWarnings reports configurations that are perfectly valid but destroy
// files in ways people rarely intend.
//
// These are warnings on purpose, and only the validate command emits them. The
// config file is the source of truth: a run never second-guesses what was
// declared, never prompts, and never refuses. The place to find out that a rule
// eats data is here, before the run — not afterwards, from the missing files.
func configWarnings(rawYAML []byte) []spec.Violation {
	var doc map[string]any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return nil // the config does not even parse; real errors cover it
	}
	cats, ok := doc["categories"].([]any)
	if !ok {
		return nil
	}
	defaults := mapAt(mapAt(doc, "configuration"), "defaults")

	var out []spec.Violation
	for i, item := range cats {
		cat, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, categoryWarnings(fmt.Sprintf("categories[%d]", i), cat, defaults)...)
	}
	return append(out, overlappingSourceWarnings(cats)...)
}

// categoryWarnings runs the per-category data-loss checks.
func categoryWarnings(prefix string, cat, defaults map[string]any) []spec.Violation {
	dst := mapAt(cat, "destination")
	src := mapAt(cat, "source")

	strategy := effectiveValue(strAt(dst, "conflict-strategy"), strAt(defaults, "conflict-strategy"), string(models.ConflictStrategyRename))
	action := effectiveValue(strAt(dst, "action"), strAt(defaults, "action"), string(models.ActionMove))
	organizeBy := effectiveValue(strAt(dst, "organize-by"), strAt(defaults, "organize-by"), "")
	rename := strAt(dst, "rename")

	var out []spec.Violation
	overwrites := strategy == string(models.ConflictStrategyOverwrite)

	// 1. A rename template where nothing varies per file gives every file the
	//    same name. With overwrite, each file destroys the one before it and
	//    only the last survives.
	if overwrites && rename != "" && !tokens.VariesPerFile(rename) {
		out = append(out, spec.Violation{
			Path: prefix + ".destination.rename",
			Message: fmt.Sprintf("every file resolves to the same name (%q has no per-file token) and conflict-strategy is %q, "+
				"so each file overwrites the previous one and only the last is kept — add {name}, {seq}, or another per-file token",
				rename, strategy),
		})
	}

	// 2. Source and destination pointing at the same directory.
	if s, d := strAt(src, "path"), strAt(dst, "path"); s != "" && d != "" && samePath(s, d) {
		out = append(out, spec.Violation{
			Path:    prefix + ".destination.path",
			Message: fmt.Sprintf("source and destination are the same directory (%s), so files are processed onto themselves", d),
		})
	}

	// 3. Archiving that deletes the originals, with no way back: undo refuses
	//    archive batches, so the only copy left is inside the archive.
	if action == string(models.ActionArchive) {
		arc := mapAt(dst, "archive")
		if keep, ok := boolAt(arc, "keep-source"); ok && !keep {
			out = append(out, spec.Violation{
				Path: prefix + ".destination.archive.keep-source",
				Message: "the original files are deleted after archiving and archive batches cannot be undone, " +
					"so the archive becomes the only copy",
			})
		}
	}

	// 4. Overwrite with nothing to separate the files: everything lands in one
	//    directory under its original name, and same-named files collide.
	if overwrites && organizeBy == "" && rename == "" {
		out = append(out, spec.Violation{
			Path: prefix + ".destination.conflict-strategy",
			Message: "all files land directly in the destination with their original names and conflict-strategy is \"overwrite\", " +
				"so any existing file with a matching name is replaced without a trace",
		})
	}
	return out
}

// overlappingSourceWarnings reports category pairs that read the same source
// directory and compete for the same extensions. The first matching category
// wins and the later one silently gets nothing, which is defined behaviour but
// almost never what was intended.
func overlappingSourceWarnings(cats []any) []spec.Violation {
	type entry struct {
		index int
		name  string
		path  string
		exts  map[string]bool
	}
	entries := make([]entry, 0, len(cats))
	for i, item := range cats {
		cat, ok := item.(map[string]any)
		if !ok {
			continue
		}
		src := mapAt(cat, "source")
		path := strAt(src, "path")
		if path == "" {
			continue
		}
		exts := make(map[string]bool)
		for _, e := range sliceAt(src, "extensions") {
			if s, ok := e.(string); ok {
				exts[strings.ToLower(s)] = true
			}
		}
		entries = append(entries, entry{index: i, name: strAt(cat, "name"), path: path, exts: exts})
	}

	var out []spec.Violation
	for a := 0; a < len(entries); a++ {
		for b := a + 1; b < len(entries); b++ {
			if !samePath(entries[a].path, entries[b].path) {
				continue
			}
			shared := sharedExtensions(entries[a].exts, entries[b].exts)
			if len(shared) == 0 {
				continue
			}
			out = append(out, spec.Violation{
				Path: fmt.Sprintf("categories[%d].source", entries[b].index),
				Message: fmt.Sprintf("competes with category %q for %s in %s — the first matching category wins, so this one may never see those files",
					entries[a].name, strings.Join(shared, ", "), entries[b].path),
			})
		}
	}
	return out
}

// sharedExtensions returns the extensions claimed by both categories, sorted for
// a stable message. The "all" sentinel matches everything, so it overlaps with
// any non-empty extension list.
func sharedExtensions(a, b map[string]bool) []string {
	if a[filters.ExtAll] && len(b) > 0 {
		return []string{"all extensions"}
	}
	if b[filters.ExtAll] && len(a) > 0 {
		return []string{"all extensions"}
	}
	var shared []string
	for ext := range a {
		if b[ext] {
			shared = append(shared, "."+ext)
		}
	}
	sort.Strings(shared)
	return shared
}

// effectiveValue resolves a setting through the category value, then the
// configuration.defaults value, then the built-in default.
func effectiveValue(categoryValue, defaultValue, builtIn string) string {
	if categoryValue != "" {
		return categoryValue
	}
	if defaultValue != "" {
		return defaultValue
	}
	return builtIn
}

// samePath reports whether two configured paths point at the same directory,
// after tilde expansion. Comparison is case-insensitive so a Windows config
// mixing "C:\Users" and "c:\users" is still recognised as one directory.
func samePath(a, b string) bool {
	return strings.EqualFold(
		filepath.Clean(config.ExpandTilde(a)),
		filepath.Clean(config.ExpandTilde(b)),
	)
}

func mapAt(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func strAt(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func boolAt(m map[string]any, key string) (value, present bool) {
	if m == nil {
		return false, false
	}
	b, ok := m[key].(bool)
	return b, ok
}

func sliceAt(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	s, _ := m[key].([]any)
	return s
}
