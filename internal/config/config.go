package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/filters"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
)

// MaxFilterNestingDepth is the maximum absolute nesting depth for CategoryFilter
// (any/all/not recursion). Used as the depth guard in validateFilter.
// The edit command's SchemaRecursionDepth uses MaxFilterNestingDepth-1 because
// yedit counts extra visits beyond the first: 1 + (N-1) = N total levels,
// matching this validation limit exactly.
const MaxFilterNestingDepth = 10

// ErrConfigNotFound is returned by InitConfig when the config file cannot be located.
var ErrConfigNotFound = errors.New("config file not found")

// InitConfig reads the YAML file at path, resolves any import: entries,
// and loads the merged document into k. Returns ErrConfigNotFound when
// the file does not exist, or a descriptive error for any other failure.
func InitConfig(k *koanf.Koanf, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%w: %w", ErrConfigNotFound, err)
	}

	merged, err := ResolveImports(path)
	if err != nil {
		return err
	}

	return k.Load(rawbytes.Provider(merged), yaml.Parser())
}

// UnmarshalConfig reads categories from k, validates them, and pre-compiles
// regex patterns. Returns an error if any category is misconfigured.
func UnmarshalConfig(k *koanf.Koanf) ([]*models.Category, error) {
	var categories []*models.Category
	if err := k.UnmarshalWithConf("categories", &categories, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return nil, fmt.Errorf("unable to decode categories: %w", err)
	}

	for _, cat := range categories {
		if err := validateCategory(cat); err != nil {
			return nil, err
		}
	}

	return categories, nil
}

// applyCategoryDefaults fills each category's empty destination fields from the
// global defaults block. Per-category values always take precedence. The default
// values are validated here, since they bypass the per-category validation that
// runs during unmarshalling.
func applyCategoryDefaults(cats []*models.Category, d *models.Defaults) error {
	if d == nil {
		return nil
	}
	if !validConflictStrategies[d.ConflictStrategy] {
		return fmt.Errorf("defaults: invalid conflict-strategy %q", d.ConflictStrategy)
	}
	if !validActions[d.Action] {
		return fmt.Errorf("defaults: invalid action %q", d.Action)
	}
	if d.OrganizeBy != "" {
		if err := tokens.ValidateTemplate(d.OrganizeBy); err != nil {
			return fmt.Errorf("defaults: invalid organize-by template: %w", err)
		}
		if tok := tokens.RenameOnlyToken(d.OrganizeBy); tok != "" {
			return fmt.Errorf("defaults: %s is not valid in organize-by; use it in rename only", tok)
		}
	}

	for _, cat := range cats {
		if cat.Destination.ConflictStrategy == "" {
			cat.Destination.ConflictStrategy = d.ConflictStrategy
		}
		if cat.Destination.Action == "" {
			cat.Destination.Action = d.Action
		}
		if cat.Destination.OrganizeBy == "" {
			cat.Destination.OrganizeBy = d.OrganizeBy
		}
	}
	return nil
}

// validActions is the set of accepted values for destination.action.
var validActions = map[models.Action]bool{
	"":                   true, // empty = default (move)
	models.ActionMove:    true,
	models.ActionCopy:    true,
	models.ActionSymlink: true,
}

// validConflictStrategies is the set of accepted values for destination.conflict-strategy.
var validConflictStrategies = map[models.ConflictStrategy]bool{
	"":                               true, // empty = default (rename)
	models.ConflictStrategyRename:    true,
	models.ConflictStrategyHashCheck: true,
	models.ConflictStrategyOverwrite: true,
	models.ConflictStrategySkip:      true,
	models.ConflictStrategyNewest:    true,
	models.ConflictStrategyOldest:    true,
	models.ConflictStrategyLarger:    true,
	models.ConflictStrategySmaller:   true,
}

// validateCategory validates a single category and pre-compiles its filter.
func validateCategory(cat *models.Category) error {
	if cat.Name == "" {
		return fmt.Errorf("category name must not be empty")
	}
	if len(cat.Source.Extensions) == 0 {
		return fmt.Errorf("category %q: source.extensions are required", cat.Name)
	}

	if cat.Source.Path != "" && cat.Destination.Path != "" &&
		filepath.Clean(cat.Source.Path) == filepath.Clean(cat.Destination.Path) {
		return fmt.Errorf("category %q: source and destination must be different directories", cat.Name)
	}

	if !validActions[cat.Destination.Action] {
		return fmt.Errorf("category %q: invalid action %q - must be move, copy, or symlink", cat.Name, cat.Destination.Action)
	}

	if !validConflictStrategies[cat.Destination.ConflictStrategy] {
		return fmt.Errorf("category %q: invalid conflict-strategy %q - must be one of: rename, hash_check, overwrite, skip, newest, oldest, larger, smaller", cat.Name, cat.Destination.ConflictStrategy)
	}

	if cat.Destination.Rename != "" {
		if err := tokens.ValidateTemplate(cat.Destination.Rename); err != nil {
			return fmt.Errorf("category %q: invalid rename template: %w", cat.Name, err)
		}
	}

	if cat.Destination.OrganizeBy != "" {
		if err := tokens.ValidateTemplate(cat.Destination.OrganizeBy); err != nil {
			return fmt.Errorf("category %q: invalid organize-by template: %w", cat.Name, err)
		}
		if tok := tokens.RenameOnlyToken(cat.Destination.OrganizeBy); tok != "" {
			return fmt.Errorf("category %q: %s is not valid in organize-by; use it in rename only", cat.Name, tok)
		}
	}

	if err := validateHooks(cat.Name, cat.Hooks); err != nil {
		return err
	}

	return validateFilter(cat.Name, &cat.Source.Filter)
}

// validateHooks validates both before and after hooks for a category.
func validateHooks(catName string, hooks *models.CategoryHooks) error {
	if hooks == nil {
		return nil
	}
	if hooks.Before != nil {
		if err := validateHook(catName, "before", hooks.Before); err != nil {
			return err
		}
	}
	if hooks.After != nil {
		if err := validateHook(catName, "after", hooks.After); err != nil {
			return err
		}
	}
	return nil
}

// validateHook validates a single CategoryHook.
func validateHook(catName, position string, hook *models.CategoryHook) error {
	if len(hook.Run) == 0 {
		return fmt.Errorf("category %q: hooks.%s.run must not be empty", catName, position)
	}
	if hook.OnFailure != "abort" && hook.OnFailure != "warn" {
		return fmt.Errorf("category %q: hooks.%s.on-failure must be \"abort\" or \"warn\"", catName, position)
	}
	return nil
}

// hasDirectFilterFields reports whether f has any direct leaf fields set.
// not is excluded: it is a modifier that can coexist with any/all.
func hasDirectFilterFields(f *models.CategoryFilter) bool {
	return f.Match != nil || f.Age != nil || f.Size != nil
}

// validateFilter validates a filter node recursively.
func validateFilter(catName string, f *models.CategoryFilter) error {
	if !FilterDepthOK(f, MaxFilterNestingDepth, 0) {
		return fmt.Errorf("category %q: filter nesting exceeds maximum depth of %d", catName, MaxFilterNestingDepth)
	}
	return validateFilterDepth(catName, f, 0)
}

// FilterDepthOK reports whether f's any/all/not nesting stays within max
// levels. depth is the level being checked (0 = the filter itself). Exported
// so the edit command's validators (internal/cmd/edit_validators.go) can
// enforce the same rule inside the TUI, without duplicating the recursion.
func FilterDepthOK(f *models.CategoryFilter, max, depth int) bool {
	if depth >= max {
		return false
	}
	for i := range f.Not {
		if !FilterDepthOK(&f.Not[i], max, depth+1) {
			return false
		}
	}
	for i := range f.Any {
		if !FilterDepthOK(&f.Any[i], max, depth+1) {
			return false
		}
	}
	for i := range f.All {
		if !FilterDepthOK(&f.All[i], max, depth+1) {
			return false
		}
	}
	return true
}

func validateFilterDepth(catName string, f *models.CategoryFilter, depth int) error {
	hasAny := len(f.Any) > 0
	hasAll := len(f.All) > 0

	if f.Any != nil && !hasAny {
		return fmt.Errorf("category %q: filter 'any' must have at least one entry", catName)
	}
	if f.All != nil && !hasAll {
		return fmt.Errorf("category %q: filter 'all' must have at least one entry", catName)
	}
	if hasAny && hasAll {
		return fmt.Errorf("category %q: filter cannot have both 'any' and 'all' at the same level", catName)
	}
	if (hasAny || hasAll) && hasDirectFilterFields(f) {
		return fmt.Errorf("category %q: filter cannot mix 'any'/'all' with direct fields", catName)
	}

	for i := range f.Not {
		if err := validateFilterDepth(catName, &f.Not[i], depth+1); err != nil {
			return err
		}
	}

	if hasAny {
		for i := range f.Any {
			if err := validateFilterDepth(catName, &f.Any[i], depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if hasAll {
		for i := range f.All {
			if err := validateFilterDepth(catName, &f.All[i], depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	return validateLeafFilter(catName, f)
}

// validateLeafFilter validates a plain (non-composite) filter node.
func validateLeafFilter(catName string, f *models.CategoryFilter) error {
	if f.Match != nil {
		if err := validateMatchFilter(catName, f.Match); err != nil {
			return err
		}
	}
	if f.Age != nil {
		if err := validateAgeFilter(catName, f.Age); err != nil {
			return err
		}
	}
	if f.Size != nil {
		if err := validateSizeFilter(catName, f.Size); err != nil {
			return err
		}
	}
	return nil
}

// validateMatchFilter validates a MatchFilter: mutually exclusive fields, valid glob, compiled regex.
func validateMatchFilter(catName string, m *models.MatchFilter) error {
	matchTypes := 0
	if m.Regex != "" {
		matchTypes++
	}
	if m.Glob != "" {
		matchTypes++
	}
	if m.Literal != "" {
		matchTypes++
	}
	if matchTypes > 1 {
		return fmt.Errorf("category %q: match.regex, match.glob, and match.literal are mutually exclusive", catName)
	}
	if m.Glob != "" {
		if err := filters.ValidateGlob(m.Glob); err != nil {
			return fmt.Errorf("category %q: %w", catName, err)
		}
	}
	return compileMatchRegex(catName, m)
}

// compileMatchRegex compiles m.Regex into m.CompiledRegex, adding (?i) when not case-sensitive.
func compileMatchRegex(catName string, m *models.MatchFilter) error {
	if m.Regex == "" {
		return nil
	}
	pattern := m.Regex
	if !m.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex in category %q: %w", catName, err)
	}
	m.CompiledRegex = compiled
	return nil
}

// validateAgeFilter checks that age.min <= age.max.
func validateAgeFilter(catName string, a *models.AgeFilter) error {
	if a.Min != 0 && a.Max != 0 && a.Min > a.Max {
		return fmt.Errorf("category %q: age.min (%s) must be less than age.max (%s)", catName, a.Min, a.Max)
	}
	return nil
}

// validateSizeFilter parses size strings and checks that size.min <= size.max.
func validateSizeFilter(catName string, s *models.SizeFilter) error {
	if s.Min != "" {
		b, err := filters.ParseSize(s.Min)
		if err != nil {
			return fmt.Errorf("category %q: invalid size.min %q: %w", catName, s.Min, err)
		}
		s.MinBytes = b
	}
	if s.Max != "" {
		b, err := filters.ParseSize(s.Max)
		if err != nil {
			return fmt.Errorf("category %q: invalid size.max %q: %w", catName, s.Max, err)
		}
		s.MaxBytes = b
	}
	if s.Min != "" && s.Max != "" && s.MinBytes > s.MaxBytes {
		return fmt.Errorf("category %q: size.min (%s) must be less than size.max (%s)", catName, s.Min, s.Max)
	}
	return nil
}

// ResolveConfigPath returns the absolute path to the config file.
// If configPath is provided it is used directly (after verifying existence).
// Otherwise it searches for movelooper.yaml in the executable directory and
// its conf/ subdirectory, returning ErrConfigNotFound if neither exists.
func ResolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return "", fmt.Errorf("resolving config path: %w", err)
		}
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %w", ErrConfigNotFound, err)
		}
		return abs, nil
	}

	homeDir, _ := os.UserHomeDir()

	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable: %v", err)
	}
	exDir := filepath.Dir(ex)

	candidates := []string{
		filepath.Join(homeDir, ".movelooper", "conf", "movelooper.yaml"),
		filepath.Join(exDir, "movelooper.yaml"),
		filepath.Join(exDir, "conf", "movelooper.yaml"),
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("%w: movelooper.yaml not found in ~/.movelooper/conf, %s, or %s/conf", ErrConfigNotFound, exDir, exDir)
}
