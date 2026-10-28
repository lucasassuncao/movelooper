package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/metadata"
	"github.com/lucasassuncao/yedit/report"
	"github.com/lucasassuncao/yedit/spec"
	"github.com/lucasassuncao/yedit/validate"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type validateFormat string

const (
	formatPretty validateFormat = "pretty"
	formatPlain  validateFormat = "plain"
	formatTable  validateFormat = "table"
	formatJSON   validateFormat = "json"
)

var validFormats = []string{string(formatPretty), string(formatPlain), string(formatTable), string(formatJSON)}

// warningNote explains, on the summary line, what a movelooper warning means.
// The rendering itself lives in yedit/report; only this sentence is ours.
const warningNote = "valid configuration that can lose files"

// ValidateCmd defines the "validate" subcommand.
func ValidateCmd() *cobra.Command {
	var (
		format    string
		summary   bool
		strict    bool
		themeName string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a configuration file and report all errors",
		// Override root's PersistentPreRunE — validate reads the file directly
		// and must not abort when the config has errors.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedTheme, err := theme.Lookup(themeName)
			if err != nil {
				return fmt.Errorf("%w: run 'movelooper --list-themes' to see available themes", err)
			}
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			return runValidate(configPath, validateFormat(format), summary, strict, selectedTheme)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "pretty", fmt.Sprintf("Output format: %s", strings.Join(validFormats, ", ")))
	cmd.Flags().BoolVar(&summary, "summary", false, "Show only error counts, not individual violations")
	cmd.Flags().BoolVar(&strict, "strict", false, "Also verify that source and destination directories exist on disk")
	cmd.Flags().StringVar(&themeName, "theme", "plain", "Theme for the pretty and table formats (same names as 'movelooper edit --theme')")
	return cmd
}

// runValidate loads the config, runs the validators, and prints the result in
// the requested format. Warnings are reported alongside errors but never fail
// the command: they flag valid configurations that lose files, and the config
// stays the source of truth about what should happen.
func runValidate(configPath string, format validateFormat, summaryOnly, strict bool, t theme.Theme) error {
	switch format {
	case formatPretty, formatPlain, formatTable, formatJSON:
	default:
		return fmt.Errorf("unknown format %q — use one of: %s", format, strings.Join(validFormats, ", "))
	}

	path, err := config.ResolveConfigPath(configPath)
	if err != nil {
		return err
	}

	violations, err := collectViolations(path, strict)
	if err != nil {
		return err
	}

	opts := report.Options{SummaryOnly: summaryOnly, Theme: t, WarningNote: warningNote}
	switch format {
	case formatJSON:
		if err := report.JSON(os.Stdout, violations, opts); err != nil {
			return fmt.Errorf("writing JSON report: %w", err)
		}
	case formatTable:
		report.Table(os.Stdout, violations, opts)
	case formatPlain:
		report.Plain(os.Stdout, violations, opts)
	default:
		report.Pretty(os.Stdout, violations, opts)
	}

	if errs, _ := report.Partition(violations); len(errs) > 0 {
		return errors.New("validation failed")
	}
	return nil
}

// collectViolations runs every rule against the config at path.
func collectViolations(path string, strict bool) ([]spec.Violation, error) {
	doc, err := document.Load(path, nil)
	if err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", path, err)
	}

	hints, err := buildMovelooperHints()
	if err != nil {
		return nil, fmt.Errorf("building hint source: %w", err)
	}

	// validate.Wire discovers the schema without linking the TUI into this
	// build, and resolves it exactly as editor.Wire does, so the rules enforced
	// here and the ones enforced while editing cannot drift apart.
	rules := append(slices.Clone(MovelooperValidators), importsResolve(path))
	wired := validate.Wire(rules, &models.Config{}, config.MaxFilterNestingDepth-1, hints)

	violations := validate.RunAll(wired, doc.Raw(), doc.Blocks())
	if strict {
		violations = append(violations, strictDirViolations(doc.Raw(), filepath.Dir(path))...)
	}
	violations = append(violations, configWarnings(doc.Raw())...)
	imported, err := importViolations(path, strict)
	if err != nil {
		return nil, err
	}
	return append(violations, imported...), nil
}

// importViolations runs the rules on every file the config imports, as the
// editor does in an import's tab; each path starts with that file's name.
func importViolations(path string, strict bool) ([]spec.Violation, error) {
	hints, err := metadata.New(models.ImportFile{})
	if err != nil {
		return nil, fmt.Errorf("building hint source: %w", err)
	}
	var out []spec.Violation
	for _, abs := range importedFiles(path) {
		name, _ := importPath(abs, path)
		name = strings.TrimPrefix(name, "./")
		doc, err := document.Load(abs, nil)
		if err != nil {
			out = append(out, spec.Violation{Path: name, Message: err.Error()})
			continue
		}
		rules := append(slices.Clone(MovelooperValidators), importsResolve(abs))
		wired := validate.Wire(rules, &models.ImportFile{}, config.MaxFilterNestingDepth-1, hints)
		found := validate.RunAll(wired, doc.Raw(), doc.Blocks())
		if strict {
			found = append(found, strictDirViolations(doc.Raw(), filepath.Dir(abs))...)
		}
		for _, v := range append(found, configWarnings(doc.Raw())...) {
			v.Path = strings.TrimSuffix(name+": "+v.Path, ": ")
			out = append(out, v)
		}
	}
	return out, nil
}

// importedFiles lists every file path imports, directly or through another
// import, once each. Entries that do not load are left to importsResolve.
func importedFiles(path string) []string {
	seen := map[string]bool{path: true}
	var out []string
	queue := []string{path}
	for len(queue) > 0 {
		file := queue[0]
		queue = queue[1:]
		raw, err := os.ReadFile(file) //#nosec G304 -- a config file the user named, or one it imports
		if err != nil {
			continue
		}
		var doc struct {
			Import []string `yaml:"import"`
		}
		if yaml.Unmarshal(raw, &doc) != nil {
			continue
		}
		for _, imp := range doc.Import {
			abs, err := config.ImportPath(file, imp)
			if err != nil || seen[abs] {
				continue
			}
			seen[abs] = true
			if _, err := os.Stat(abs); err == nil {
				out = append(out, abs)
				queue = append(queue, abs)
			}
		}
	}
	return out
}

// strictDirViolations checks whether source.path and destination.path for each
// category exist on disk, with relative paths read from dir, the file's folder.
func strictDirViolations(rawYAML []byte, dir string) []spec.Violation {
	var doc map[string]any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return nil
	}
	cats, ok := doc["categories"].([]any)
	if !ok {
		return nil
	}
	var out []spec.Violation
	for i, item := range cats {
		cat, ok := item.(map[string]any)
		if !ok {
			continue
		}
		prefix := fmt.Sprintf("categories[%d]", i)
		if src, ok := cat["source"].(map[string]any); ok {
			if p, ok := src["path"].(string); ok && p != "" {
				if _, err := os.Stat(config.ExpandTilde(config.AnchorPath(p, dir))); os.IsNotExist(err) {
					out = append(out, spec.Violation{
						Path:    prefix + ".source.path",
						Message: fmt.Sprintf("directory does not exist: %s", p),
					})
				}
			}
		}
		if dst, ok := cat["destination"].(map[string]any); ok {
			if p, ok := dst["path"].(string); ok && p != "" {
				if _, err := os.Stat(config.ExpandTilde(config.AnchorPath(p, dir))); os.IsNotExist(err) {
					out = append(out, spec.Violation{
						Path:    prefix + ".destination.path",
						Message: fmt.Sprintf("directory does not exist: %s", p),
					})
				}
			}
		}
	}
	return out
}
