package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

// ValidateCmd defines the "validate" subcommand.
func ValidateCmd() *cobra.Command {
	var (
		format  string
		summary bool
		strict  bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a configuration file and report all errors",
		// Override root's PersistentPreRunE — validate reads the file directly
		// and must not abort when the config has errors.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			return runValidate(configPath, validateFormat(format), summary, strict)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "pretty", fmt.Sprintf("Output format: %s", strings.Join(validFormats, ", ")))
	cmd.Flags().BoolVar(&summary, "summary", false, "Show only error counts, not individual violations")
	cmd.Flags().BoolVar(&strict, "strict", false, "Also verify that source and destination directories exist on disk")
	return cmd
}

// runValidate performs the validation logic: it loads the config, runs validators, and prints results in the specified format.
func runValidate(configPath string, format validateFormat, summaryOnly bool, strict bool) error {
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
	wired := editor.Wire(MovelooperValidators, editor.Config{
		Schema:               &models.Config{},
		Metadata:             hints,
		SchemaRecursionDepth: config.MaxFilterNestingDepth - 1,
	})

	violations := editor.RunAll(wired, doc.Raw(), doc.Blocks())

	if strict {
		violations = append(violations, strictDirViolations(doc.Raw())...)
	}

	switch format {
	case formatJSON:
		printValidateJSON(violations, summaryOnly)
	case formatTable:
		printValidateTable(violations, summaryOnly)
	case formatPlain:
		printValidatePlain(violations, summaryOnly)
	default:
		printValidatePretty(violations, summaryOnly)
	}

	if len(violations) > 0 {
		return errors.New("validation failed")
	}
	return nil
}

// strictDirViolations checks whether source.path and destination.path for each
// category exist on disk, returning a violation for each path that does not.
func strictDirViolations(rawYAML []byte) []editor.Violation {
	var doc map[string]any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return nil
	}
	cats, ok := doc["categories"].([]any)
	if !ok {
		return nil
	}
	var out []editor.Violation
	for i, item := range cats {
		cat, ok := item.(map[string]any)
		if !ok {
			continue
		}
		prefix := fmt.Sprintf("categories[%d]", i)
		if src, ok := cat["source"].(map[string]any); ok {
			if p, ok := src["path"].(string); ok && p != "" {
				if _, err := os.Stat(p); os.IsNotExist(err) {
					out = append(out, editor.Violation{
						Path:    prefix + ".source.path",
						Message: fmt.Sprintf("directory does not exist: %s", p),
					})
				}
			}
		}
		if dst, ok := cat["destination"].(map[string]any); ok {
			if p, ok := dst["path"].(string); ok && p != "" {
				if _, err := os.Stat(p); os.IsNotExist(err) {
					out = append(out, editor.Violation{
						Path:    prefix + ".destination.path",
						Message: fmt.Sprintf("directory does not exist: %s", p),
					})
				}
			}
		}
	}
	return out
}

var topSectionRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)`)

// sectionOf extracts the top-level section name from a violation path, or returns "(general)" if it cannot be determined.
func sectionOf(path string) string {
	if m := topSectionRe.FindString(path); m != "" {
		return m
	}
	return "(general)"
}

// groupViolations groups violations by their top-level section and returns
// sections in a stable order (general → alphabetical).
func groupViolations(violations []editor.Violation) ([]string, map[string][]editor.Violation) {
	bySection := make(map[string][]editor.Violation)
	for _, v := range violations {
		s := sectionOf(v.Path)
		bySection[s] = append(bySection[s], v)
	}
	sections := make([]string, 0, len(bySection))
	for s := range bySection {
		sections = append(sections, s)
	}
	sort.Strings(sections)
	return sections, bySection
}

// subPath strips the top-level section prefix from a violation path.
func subPath(path string) string {
	if i := strings.IndexAny(path, ".["); i >= 0 {
		rest := path[i:]
		return strings.TrimPrefix(rest, ".")
	}
	return path
}

// summaryLine builds the coloured summary string shared by pretty and table formats.
func summaryLine(sections []string, bySection map[string][]editor.Violation) string {
	parts := make([]string, 0, len(sections))
	total := 0
	for _, s := range sections {
		n := len(bySection[s])
		total += n
		parts = append(parts, fmt.Sprintf("%d in %s", n, s))
	}
	return fmt.Sprintf("%s error(s) — %s",
		pterm.Red(fmt.Sprintf("%d", total)),
		strings.Join(parts, ", "),
	)
}

// printValidatePretty renders violations grouped by section with a tree-like structure and summary.
func printValidatePretty(violations []editor.Violation, summaryOnly bool) {
	if len(violations) == 0 {
		pterm.Success.Println("No errors found — configuration is valid")
		return
	}

	sections, bySection := groupViolations(violations)

	if !summaryOnly {
		for _, section := range sections {
			vs := bySection[section]
			pterm.Println()
			pterm.Bold.Println("  " + section)
			for i, v := range vs {
				connector := pterm.Gray("├─")
				if i == len(vs)-1 {
					connector = pterm.Gray("└─")
				}
				sp := pterm.Yellow(fmt.Sprintf("%-44s", subPath(v.Path)))
				pterm.Printf("  %s %s %s\n", connector, sp, v.Message)
			}
		}
		pterm.Println()
	}

	pterm.Println(summaryLine(sections, bySection))
}

// printValidatePlain renders violations as plain text lines with a final count.
func printValidatePlain(violations []editor.Violation, summaryOnly bool) {
	if len(violations) == 0 {
		fmt.Println("ok")
		return
	}

	if !summaryOnly {
		for _, v := range violations {
			fmt.Printf("%-48s %s\n", v.Path, v.Message)
		}
	}
	fmt.Printf("%d error(s)\n", len(violations))
}

// printValidateTable renders violations in tables grouped by section, with a final summary line.
func printValidateTable(violations []editor.Violation, summaryOnly bool) {
	if len(violations) == 0 {
		pterm.Success.Println("No errors found — configuration is valid")
		return
	}

	sections, bySection := groupViolations(violations)

	if !summaryOnly {
		termWidth, _, err := term.GetSize(int(os.Stdout.Fd())) //#nosec G115 -- fd is always a small positive number
		if err != nil || termWidth < 60 {
			termWidth = 120
		}

		for _, section := range sections {
			vs := bySection[section]
			pterm.Println()
			pterm.Bold.Printf("%s", section)
			pterm.Printf("  %s\n", pterm.Gray(fmt.Sprintf("(%d errors)", len(vs))))
			renderSectionTable(vs, termWidth)
		}
		pterm.Println()
	}

	pterm.Println(summaryLine(sections, bySection))
}

// renderSectionTable renders a single section's violations in a table, adjusting column widths to fit the terminal.
func renderSectionTable(vs []editor.Violation, termWidth int) {
	const col1Max = 40
	const borders = 7 // "│ " + " │ " + " │"
	col2Max := termWidth - col1Max - borders
	if col2Max < 30 {
		col2Max = 30
	}

	t := prettytable.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(prettytable.StyleRounded)
	t.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, WidthMax: col1Max, Colors: text.Colors{text.FgYellow}},
		{Number: 2, WidthMax: col2Max},
	})
	t.AppendHeader(prettytable.Row{"PATH", "ERROR"})
	for _, v := range vs {
		t.AppendRow(prettytable.Row{subPath(v.Path), v.Message})
	}
	t.Render()
}

// validateJSONOutput defines the structure of the JSON output for validation results.
type validateJSONOutput struct {
	Valid      bool                    `json:"valid"`
	ErrorCount int                     `json:"error_count"`
	Errors     []validateJSONViolation `json:"errors,omitempty"`
	Summary    map[string]int          `json:"summary"`
}

// validateJSONViolation defines the structure of individual violations in the JSON output.
type validateJSONViolation struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// printValidateJSON renders violations as a JSON object with overall validity, error count, optional details, and a summary by section.
func printValidateJSON(violations []editor.Violation, summaryOnly bool) {
	_, bySection := groupViolations(violations)

	summary := make(map[string]int, len(bySection))
	for s, vs := range bySection {
		summary[s] = len(vs)
	}

	out := validateJSONOutput{
		Valid:      len(violations) == 0,
		ErrorCount: len(violations),
		Summary:    summary,
	}

	if !summaryOnly {
		out.Errors = make([]validateJSONViolation, len(violations))
		for i, v := range violations {
			out.Errors[i] = validateJSONViolation{Path: v.Path, Message: v.Message}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
