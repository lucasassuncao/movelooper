package cmd

import (
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testParseCategoryNames defines the structure for test cases of the ParseCategoryNames function,
// containing the test name, the raw input string, and the expected slice of names.
type testParseCategoryNames struct {
	name  string
	input string
	want  []string
}

// testParseCategoryNamesTestCases defines a set of test cases for the ParseCategoryNames function,
// covering empty input, single name, multiple names, whitespace trimming, and separator-only input.
var testParseCategoryNamesTestCases = []testParseCategoryNames{
	{"empty input returns nil", "", nil},
	{"single name", "images", []string{"images"}},
	{"multiple names", "images,docs", []string{"images", "docs"}},
	{"whitespace is trimmed", " images , docs ", []string{"images", "docs"}},
	{"separator only returns nil", ",", nil},
}

// TestParseCategoryNames tests the ParseCategoryNames function with various input strings
// to ensure it correctly parses comma-separated category names.
func TestParseCategoryNames(t *testing.T) {
	t.Parallel()
	for _, tt := range testParseCategoryNamesTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ParseCategoryNames(tt.input))
		})
	}
}

// testFilterCategories defines the structure for test cases of the FilterCategories function,
// containing the filter names, includeDisabled flag, expected category names, and an optional error substring.
type testFilterCategories struct {
	name            string
	names           []string
	includeDisabled bool
	wantNames       []string
	wantErr         string
}

// testFilterCategoriesTestCases defines a set of test cases for the FilterCategories function,
// covering empty names, includeDisabled flag, single and multiple names, unknown names,
// disabled categories, and mixed valid/unknown names.
var testFilterCategoriesTestCases = []testFilterCategories{
	{
		name:      "empty names returns all enabled",
		names:     nil,
		wantNames: []string{"images", "docs"},
	},
	{
		name:            "empty names with includeDisabled returns all",
		names:           nil,
		includeDisabled: true,
		wantNames:       []string{"images", "docs", "archive"},
	},
	{
		name:      "single valid name",
		names:     []string{"images"},
		wantNames: []string{"images"},
	},
	{
		name:      "multiple valid names",
		names:     []string{"images", "docs"},
		wantNames: []string{"images", "docs"},
	},
	{
		name:    "unknown name returns error",
		names:   []string{"unknown"},
		wantErr: `unknown category "unknown"`,
	},
	{
		name:      "disabled category without includeDisabled is skipped",
		names:     []string{"archive"},
		wantNames: nil,
	},
	{
		name:            "disabled category with includeDisabled is included",
		names:           []string{"archive"},
		includeDisabled: true,
		wantNames:       []string{"archive"},
	},
	{
		name:    "one valid one unknown returns error",
		names:   []string{"images", "missing"},
		wantErr: `unknown category "missing"`,
	},
}

// TestFilterCategories tests the FilterCategories function with various combinations of
// category names and flags to ensure it correctly filters and validates categories.
func TestFilterCategories(t *testing.T) {
	t.Parallel()
	all := []*models.Category{
		enabledCategory("images"),
		enabledCategory("docs"),
		disabledCategory("archive"),
	}
	logger := silentLogger()

	for _, tt := range testFilterCategoriesTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FilterCategories(all, tt.names, tt.includeDisabled, logger)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			var gotNames []string
			for _, c := range result {
				gotNames = append(gotNames, c.Name)
			}
			assert.Equal(t, tt.wantNames, gotNames)
		})
	}
}

func enabledCategory(name string) *models.Category {
	t := true
	return &models.Category{Name: name, Enabled: &t}
}

func disabledCategory(name string) *models.Category {
	f := false
	return &models.Category{Name: name, Enabled: &f}
}

func silentLogger() *pterm.Logger {
	l := pterm.DefaultLogger
	l.Level = pterm.LogLevelDisabled
	return &l
}
