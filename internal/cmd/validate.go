package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lucasassuncao/bezel/theme"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/report"
	"github.com/lucasassuncao/yedit/spec"
	"github.com/lucasassuncao/yedit/validate"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
				return fmt.Errorf("%w: run 'movelooper edit --list-themes' to see available themes", err)
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

	doc, err := document.Load(path, nil)
	if err != nil {
		return fmt.Errorf("could not parse %s: %w", path, err)
	}

	hints, err := buildMovelooperHints()
	if err != nil {
		return fmt.Errorf("building hint source: %w", err)
	}

	// validate.Wire discovers the schema without linking the TUI into this
	// build, and resolves it exactly as editor.Wire does, so the rules enforced
	// here and the ones enforced while editing cannot drift apart.
	wired := validate.Wire(MovelooperValidators, &models.Config{}, config.MaxFilterNestingDepth-1, hints)

	violations := validate.RunAll(wired, doc.Raw(), doc.Blocks())
	if strict {
		violations = append(violations, strictDirViolations(doc.Raw())...)
	}
	violations = append(violations, configWarnings(doc.Raw())...)

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

// strictDirViolations checks whether source.path and destination.path for each
// category exist on disk, returning a violation for each path that does not.
func strictDirViolations(rawYAML []byte) []spec.Violation {
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
				if _, err := os.Stat(config.ExpandTilde(p)); os.IsNotExist(err) {
					out = append(out, spec.Violation{
						Path:    prefix + ".source.path",
						Message: fmt.Sprintf("directory does not exist: %s", p),
					})
				}
			}
		}
		if dst, ok := cat["destination"].(map[string]any); ok {
			if p, ok := dst["path"].(string); ok && p != "" {
				if _, err := os.Stat(config.ExpandTilde(p)); os.IsNotExist(err) {
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
