package cmd

import (
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enabledCat(name string) *models.Category {
	return buildCategory(name, "/src", "/dst", []string{"pdf"})
}

func disabledCat(name string) *models.Category {
	cat := buildCategory(name, "/src", "/dst", []string{"pdf"})
	f := false
	cat.Enabled = &f
	return cat
}

func TestParseCategoryNames(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"images", []string{"images"}},
		{"images,docs", []string{"images", "docs"}},
		{" images , docs ", []string{"images", "docs"}},
		{",", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, parseCategoryNames(tt.input))
		})
	}
}

func TestFilterCategories(t *testing.T) {
	all := []*models.Category{
		enabledCat("images"),
		enabledCat("docs"),
		disabledCat("archive"),
	}
	m := newSilentMovelooper(all)

	tests := []struct {
		name            string
		names           []string
		includeDisabled bool
		wantNames       []string
		wantErr         string
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterCategories(all, tt.names, tt.includeDisabled, m.Logger)
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
