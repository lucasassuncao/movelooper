package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalCategory = `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`

// testInitConfig defines the structure for test cases of the InitConfig function,
// containing YAML content, a non-existent path flag, an error expectation flag,
// and an optional specific error to check with errors.Is.
type testInitConfig struct {
	name        string
	yaml        string
	nonExistent bool
	wantErr     bool
	errIs       error
}

// testInitConfigTestCases defines a set of test cases for the InitConfig function,
// covering file not found, malformed YAML, valid minimal config, and empty file scenarios.
var testInitConfigTestCases = []testInitConfig{
	{"file not found", "", true, true, ErrConfigNotFound},
	{"malformed yaml", "categories: [invalid: yaml: :", false, true, nil},
	{"valid minimal config", minimalCategory, false, false, nil},
	{"empty file is valid", "", false, false, nil},
}

// TestInitConfig tests the InitConfig function to ensure it correctly loads and validates config files.
func TestInitConfig(t *testing.T) {
	for _, tt := range testInitConfigTestCases {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var path string
			if tt.nonExistent {
				path = "/nonexistent/path/movelooper.yaml"
			} else {
				path = writeYAML(t, dir, "cfg.yaml", tt.yaml)
			}

			k := koanf.New(".")
			err := InitConfig(k, path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

// testUnmarshalConfig defines the structure for test cases of the UnmarshalConfig function,
// containing YAML content, expected error substring, any-error flag, and a check function.
type testUnmarshalConfig struct {
	name       string
	yaml       string
	wantErr    string
	wantAnyErr bool
	check      func(t *testing.T, cats []*models.Category)
}

// testUnmarshalConfigTestCases defines a set of test cases for the UnmarshalConfig function,
// covering valid categories, missing extensions, invalid regex, mutually exclusive filters,
// size/age constraints, regex compilation, size bytes population, invalid glob, and hook validation.
var testUnmarshalConfigTestCases = []testUnmarshalConfig{
	{
		name: "valid category",
		yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf, txt]
    destination:
      path: /tmp/dst
`,
		check: func(t *testing.T, cats []*models.Category) {
			require.Len(t, cats, 1)
			assert.Equal(t, "docs", cats[0].Name)
			assert.ElementsMatch(t, []string{"pdf", "txt"}, cats[0].Source.Extensions)
		},
	},
	{
		name:    "missing extensions",
		wantErr: "source.extensions are required",
		yaml: `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`,
	},
	{
		name:    "invalid regex",
		wantErr: "invalid regex",
		yaml: `
categories:
  - name: bad-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        match:
          regex: "[invalid"
    destination:
      path: /tmp/dst
`,
	},
	{
		name:    "glob and literal mutually exclusive",
		wantErr: "mutually exclusive",
		yaml: `
categories:
  - name: both-filters
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        match:
          glob: "*.txt"
          literal: "report.txt"
    destination:
      path: /tmp/dst
`,
	},
	{
		name:    "min-size greater than max-size",
		wantErr: "size.min",
		yaml: `
categories:
  - name: bad-size
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        size:
          min: "10 MB"
          max: "1 MB"
    destination:
      path: /tmp/dst
`,
	},
	{
		name:    "min-age greater than max-age",
		wantErr: "age.min",
		yaml: `
categories:
  - name: bad-age
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        age:
          min: "48h"
          max: "24h"
    destination:
      path: /tmp/dst
`,
	},
	{
		name: "case-insensitive regex compiled",
		yaml: `
categories:
  - name: ci-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        match:
          regex: "report"
          case-sensitive: false
    destination:
      path: /tmp/dst
`,
		check: func(t *testing.T, cats []*models.Category) {
			require.NotNil(t, cats[0].Source.Filter.Match)
			require.NotNil(t, cats[0].Source.Filter.Match.CompiledRegex)
			assert.True(t, cats[0].Source.Filter.Match.CompiledRegex.MatchString("REPORT"))
		},
	},
	{
		name: "case-sensitive regex compiled",
		yaml: `
categories:
  - name: cs-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        match:
          regex: "report"
          case-sensitive: true
    destination:
      path: /tmp/dst
`,
		check: func(t *testing.T, cats []*models.Category) {
			require.NotNil(t, cats[0].Source.Filter.Match)
			require.NotNil(t, cats[0].Source.Filter.Match.CompiledRegex)
			assert.False(t, cats[0].Source.Filter.Match.CompiledRegex.MatchString("REPORT"))
			assert.True(t, cats[0].Source.Filter.Match.CompiledRegex.MatchString("report"))
		},
	},
	{
		name: "size bytes populated",
		yaml: `
categories:
  - name: sized
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        size:
          min: "1 KB"
          max: "10 MB"
    destination:
      path: /tmp/dst
`,
		check: func(t *testing.T, cats []*models.Category) {
			require.NotNil(t, cats[0].Source.Filter.Size)
			assert.Equal(t, int64(1000), cats[0].Source.Filter.Size.MinBytes)
			assert.Equal(t, int64(10_000_000), cats[0].Source.Filter.Size.MaxBytes)
		},
	},
	{
		name:       "invalid glob returns error",
		wantAnyErr: true,
		yaml: `
categories:
  - name: bad-glob
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        match:
          glob: "[invalid"
    destination:
      path: /tmp/dst
`,
	},
	{
		name:    "hook with empty run list is rejected",
		wantErr: `hooks.before.run must not be empty`,
		yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
    hooks:
      before:
        on-failure: abort
        run: []
`,
	},
	{
		name:    "hook with invalid on-failure is rejected",
		wantErr: `hooks.after.on-failure must be "abort" or "warn"`,
		yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
    hooks:
      after:
        on-failure: explode
        run:
          - echo done
`,
	},
	{
		name:    "hook with no on-failure is rejected",
		wantErr: `hooks.before.on-failure must be "abort" or "warn"`,
		yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
    hooks:
      before:
        run:
          - echo hi
`,
	},
	{
		name: "valid hook is accepted",
		yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
    hooks:
      before:
        on-failure: abort
        run:
          - echo starting
      after:
        on-failure: warn
        run:
          - echo done
`,
		check: func(t *testing.T, cats []*models.Category) {
			require.Len(t, cats, 1)
			require.NotNil(t, cats[0].Hooks)
			require.NotNil(t, cats[0].Hooks.Before)
			require.NotNil(t, cats[0].Hooks.After)
			assert.Equal(t, "abort", cats[0].Hooks.Before.OnFailure)
			assert.Equal(t, []string{"echo starting"}, cats[0].Hooks.Before.Run)
			assert.Equal(t, "warn", cats[0].Hooks.After.OnFailure)
		},
	},
}

// TestUnmarshalConfig tests the UnmarshalConfig function to ensure it correctly
// parses and validates category configurations from koanf.
func TestUnmarshalConfig(t *testing.T) {
	for _, tt := range testUnmarshalConfigTestCases {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeYAML(t, dir, "cfg.yaml", tt.yaml)
			k := koanf.New(".")
			require.NoError(t, InitConfig(k, path))

			cats, err := UnmarshalConfig(k)

			switch {
			case tt.wantAnyErr:
				assert.Error(t, err)
			case tt.wantErr != "":
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			default:
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, cats)
				}
			}
		})
	}
}

// testLoadConfig defines the structure for test cases of the LoadConfig function,
// containing YAML content and a check function for assertions on the resulting Configuration.
type testLoadConfig struct {
	name  string
	yaml  string
	check func(t *testing.T, cfg models.Configuration)
}

// testLoadConfigTestCases defines a set of test cases for the LoadConfig function,
// covering default values, custom values, and watch-delay fallback.
var testLoadConfigTestCases = []testLoadConfig{
	{
		name: "defaults when not set",
		yaml: "",
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
			assert.Equal(t, defaultHistoryLimit, cfg.HistoryLimit)
		},
	},
	{
		name: "custom values",
		yaml: `
configuration:
  output: json
  log-level: debug
  watch-delay: 2m
  history-limit: 100
`,
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, "json", cfg.Output)
			assert.Equal(t, "debug", cfg.LogLevel)
			assert.Equal(t, 2*time.Minute, cfg.WatchDelay)
			assert.Equal(t, 100, cfg.HistoryLimit)
		},
	},
	{
		name: "watch-delay fallback to default",
		yaml: "configuration:\n  output: text\n",
		check: func(t *testing.T, cfg models.Configuration) {
			assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
		},
	},
}

// TestLoadConfig tests the LoadConfig function to ensure it correctly applies
// defaults and parses custom configuration values.
func TestLoadConfig(t *testing.T) {
	for _, tt := range testLoadConfigTestCases {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeYAML(t, dir, "cfg.yaml", tt.yaml)
			k := koanf.New(".")
			if tt.yaml != "" {
				require.NoError(t, InitConfig(k, path))
			}
			tt.check(t, LoadConfig(k))
		})
	}
}

// testValidateCategoryAction defines the structure for test cases of the validateCategory function
// for the action field, containing the action value and an error expectation flag.
type testValidateCategoryAction struct {
	name    string
	action  string
	wantErr bool
}

// testValidateCategoryActionTestCases defines a set of test cases for validateCategory action validation,
// covering empty, valid, and invalid action values.
var testValidateCategoryActionTestCases = []testValidateCategoryAction{
	{"empty defaults to move - ok", "", false},
	{"move explicit - ok", "move", false},
	{"copy - ok", "copy", false},
	{"symlink - ok", "symlink", false},
	{"invalid action", "link", true},
	{"uppercase invalid", "MOVE", true},
}

// TestValidateCategoryAction tests validateCategory to ensure it correctly validates the action field.
func TestValidateCategoryAction(t *testing.T) {
	for _, tt := range testValidateCategoryActionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			cat := &models.Category{
				Name: "test",
				Source: models.CategorySource{
					Extensions: []string{"pdf"},
				},
				Destination: models.CategoryDestination{
					Path:   "/tmp/dst",
					Action: models.Action(tt.action),
				},
			}
			err := validateCategory(cat)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "action")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testValidateCategoryRename defines the structure for test cases of the validateCategory function
// for the rename field, containing the rename template and an error expectation flag.
type testValidateCategoryRename struct {
	name    string
	rename  string
	wantErr bool
}

// testValidateCategoryRenameTestCases defines a set of test cases for validateCategory rename validation,
// covering empty, valid, and unknown token rename templates.
var testValidateCategoryRenameTestCases = []testValidateCategoryRename{
	{"empty - ok", "", false},
	{"valid template - ok", "{mod-date}_{name}.{ext}", false},
	{"unknown token - error", "{unknown}", true},
}

// TestValidateCategoryRename tests validateCategory to ensure it correctly validates the rename field.
func TestValidateCategoryRename(t *testing.T) {
	for _, tt := range testValidateCategoryRenameTestCases {
		t.Run(tt.name, func(t *testing.T) {
			cat := &models.Category{
				Name: "test",
				Source: models.CategorySource{
					Extensions: []string{"pdf"},
				},
				Destination: models.CategoryDestination{
					Path:   "/tmp/dst",
					Rename: tt.rename,
				},
			}
			err := validateCategory(cat)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "rename")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testValidateCategoryOrganizeBy defines the structure for test cases of the validateCategory function
// for the organize-by field, containing the template, error expectation, and expected error message.
type testValidateCategoryOrganizeBy struct {
	name       string
	organizeBy string
	wantErr    bool
	errMsg     string
}

// testValidateCategoryOrganizeByTestCases defines a set of test cases for validateCategory organize-by validation,
// covering empty, valid, unknown token, and seq-in-organize-by scenarios.
var testValidateCategoryOrganizeByTestCases = []testValidateCategoryOrganizeBy{
	{"empty - ok", "", false, ""},
	{"valid tokens - ok", "{ext}/{mod-year}", false, ""},
	{"unknown token - error", "{unknown}", true, "organize-by"},
	{"seq in organize-by - error", "{seq:4}/{ext}", true, "{seq}"},
	{"seq no padding in organize-by - error", "{seq}/{ext}", true, "{seq}"},
}

// TestValidateCategoryOrganizeBy tests validateCategory to ensure it correctly validates the organize-by field.
func TestValidateCategoryOrganizeBy(t *testing.T) {
	for _, tt := range testValidateCategoryOrganizeByTestCases {
		t.Run(tt.name, func(t *testing.T) {
			cat := &models.Category{
				Name: "test",
				Source: models.CategorySource{
					Extensions: []string{"pdf"},
				},
				Destination: models.CategoryDestination{
					Path:       "/tmp/dst",
					OrganizeBy: tt.organizeBy,
				},
			}
			err := validateCategory(cat)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testValidateFilterAnyAll defines the structure for test cases of the validateFilter function
// for Any/All composite filters, containing the filter, error expectation, and expected error message.
type testValidateFilterAnyAll struct {
	name    string
	filter  models.CategoryFilter
	wantErr bool
	errMsg  string
}

// testValidateFilterAnyAllTestCases defines a set of test cases for validateFilter Any/All/Not validation,
// covering valid compositions, invalid combinations, and invalid child filters.
var testValidateFilterAnyAllTestCases = []testValidateFilterAnyAll{
	{
		name: "valid any with glob groups",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Match: &models.MatchFilter{Glob: "invoice_*"}},
			},
		},
		wantErr: false,
	},
	{
		name: "valid all with size and glob",
		filter: models.CategoryFilter{
			All: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "report_*"}},
				{Size: &models.SizeFilter{Min: "1MB"}},
			},
		},
		wantErr: false,
	},
	{
		name: "valid any inside all",
		filter: models.CategoryFilter{
			All: []models.CategoryFilter{
				{Size: &models.SizeFilter{Min: "1MB"}},
				{Any: []models.CategoryFilter{
					{Match: &models.MatchFilter{Glob: "report_*"}},
					{Match: &models.MatchFilter{Glob: "invoice_*"}},
				}},
			},
		},
		wantErr: false,
	},
	{
		name: "any and all at same level - error",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{{Match: &models.MatchFilter{Glob: "report_*"}}},
			All: []models.CategoryFilter{{Match: &models.MatchFilter{Glob: "invoice_*"}}},
		},
		wantErr: true,
		errMsg:  "cannot have both 'any' and 'all'",
	},
	{
		name: "any mixed with direct fields - error",
		filter: models.CategoryFilter{
			Match: &models.MatchFilter{Glob: "report_*"},
			Any:   []models.CategoryFilter{{Match: &models.MatchFilter{Glob: "invoice_*"}}},
		},
		wantErr: true,
		errMsg:  "cannot mix 'any'/'all' with direct fields",
	},
	{
		name: "all mixed with direct fields - error",
		filter: models.CategoryFilter{
			Size: &models.SizeFilter{Min: "1MB"},
			All:  []models.CategoryFilter{{Match: &models.MatchFilter{Glob: "report_*"}}},
		},
		wantErr: true,
		errMsg:  "cannot mix 'any'/'all' with direct fields",
	},
	{
		name: "any with invalid child - error (regex and glob both set)",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Regex: "report", Glob: "report_*"}},
			},
		},
		wantErr: true,
		errMsg:  "mutually exclusive",
	},
	{
		name: "valid any with regex in one group and glob in another",
		filter: models.CategoryFilter{
			Any: []models.CategoryFilter{
				{Match: &models.MatchFilter{Regex: `^\d{4}-.*`}},
				{Match: &models.MatchFilter{Glob: "report_*"}},
			},
		},
		wantErr: false,
	},
	{
		name: "not with valid sub-filter",
		filter: models.CategoryFilter{
			Not: []models.CategoryFilter{
				{Match: &models.MatchFilter{Glob: "*_draft*"}},
			},
		},
		wantErr: false,
	},
}

// TestValidateFilterAnyAll tests the validateFilter function with Any/All composite filters
// to ensure it correctly validates nested filter compositions.
func TestValidateFilterAnyAll(t *testing.T) {
	for _, tt := range testValidateFilterAnyAllTestCases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilter("test", &tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// writeYAML writes content to a temp file and returns its path.
func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}
