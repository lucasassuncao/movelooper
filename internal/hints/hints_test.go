package hints_test

import (
	"testing"

	"github.com/lucasassuncao/movelooper/internal/hints"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
)

// --- BuildFrom: Type derivation ---

func TestBuildFrom_derivesTypeForPrimitiveFields(t *testing.T) {
	tree := map[string]*hints.HintNode{
		"configuration": {
			Children: map[string]*hints.HintNode{
				"output":        {},
				"show-caller":   {},
				"history-limit": {},
				"watch-delay":   {},
			},
		},
	}
	src := hints.BuildFrom(&models.Config{}, tree)

	cases := []struct {
		field    string
		wantType string
	}{
		{"output", "string"},
		{"show-caller", "bool"},
		{"history-limit", "int"},
		{"watch-delay", "duration"},
	}
	for _, tc := range cases {
		meta := src.FieldHint("configuration", tc.field)
		if meta.Type != tc.wantType {
			t.Errorf("field %q: want Type=%q, got %q", tc.field, tc.wantType, meta.Type)
		}
	}
}

func TestBuildFrom_derivesTypeForSliceAndObject(t *testing.T) {
	tree := map[string]*hints.HintNode{
		"categories": {
			Children: map[string]*hints.HintNode{
				"source": {
					Children: map[string]*hints.HintNode{
						"extensions": {},
						"path":       {},
					},
				},
			},
		},
	}
	src := hints.BuildFrom(&models.Config{}, tree)

	if meta := src.FieldHint("categories", "source"); meta.Type != "object" {
		t.Errorf("source: want Type=object, got %q", meta.Type)
	}
	if meta := src.FieldHint("categories", "source.extensions"); meta.Type != "[]string" {
		t.Errorf("source.extensions: want Type=[]string, got %q", meta.Type)
	}
	if meta := src.FieldHint("categories", "source.path"); meta.Type != "string" {
		t.Errorf("source.path: want Type=string, got %q", meta.Type)
	}
}

// --- FieldHint: path resolution ---

func TestFieldHint_blockLevelHint(t *testing.T) {
	tree := map[string]*hints.HintNode{
		"configuration": {
			FieldMeta: editor.FieldMeta{Description: "General settings."},
		},
	}
	src := hints.BuildFrom(&models.Config{}, tree)
	meta := src.FieldHint("configuration", "")
	if meta.Description != "General settings." {
		t.Errorf("want Description=%q, got %q", "General settings.", meta.Description)
	}
}

func TestFieldHint_nestedPath(t *testing.T) {
	tree := map[string]*hints.HintNode{
		"configuration": {
			Children: map[string]*hints.HintNode{
				"output": {FieldMeta: editor.FieldMeta{Description: "Log destination."}},
			},
		},
	}
	src := hints.BuildFrom(&models.Config{}, tree)
	meta := src.FieldHint("configuration", "output")
	if meta.Description != "Log destination." {
		t.Errorf("want Description=%q, got %q", "Log destination.", meta.Description)
	}
}

func TestFieldHint_missingPathReturnsZero(t *testing.T) {
	tree := map[string]*hints.HintNode{
		"configuration": {Children: map[string]*hints.HintNode{}},
	}
	src := hints.BuildFrom(&models.Config{}, tree)
	meta := src.FieldHint("configuration", "nonexistent")
	if meta.Description != "" || meta.Type != "" || meta.Required || meta.Default != "" || len(meta.OneOf) != 0 || meta.Example != "" {
		t.Errorf("expected zero FieldMeta for missing path; got %+v", meta)
	}
}

func TestFieldHint_missingBlockReturnsZero(t *testing.T) {
	src := hints.BuildFrom(&models.Config{}, map[string]*hints.HintNode{})
	meta := src.FieldHint("nonexistent", "")
	if meta.Description != "" || meta.Type != "" || meta.Required || meta.Default != "" || len(meta.OneOf) != 0 || meta.Example != "" {
		t.Errorf("expected zero FieldMeta for missing block; got %+v", meta)
	}
}

// --- Recursive resolution ---

func TestFieldHint_recursiveFilter(t *testing.T) {
	// filterChildren must be built in two phases: create the "any" node first,
	// then assign Children after filterChildren is fully initialized.
	anyNode := &hints.HintNode{FieldMeta: editor.FieldMeta{Description: "OR logic."}}
	filterChildren := map[string]*hints.HintNode{
		"regex": {FieldMeta: editor.FieldMeta{Description: "RE2 regex."}},
		"any":   anyNode,
	}
	anyNode.Children = filterChildren // back-reference: any depth resolves correctly

	tree := map[string]*hints.HintNode{
		"categories": {
			Children: map[string]*hints.HintNode{
				"source": {
					Children: map[string]*hints.HintNode{
						"filter": {Children: filterChildren},
					},
				},
			},
		},
	}
	src := hints.BuildFrom(&models.Config{}, tree)

	cases := []struct {
		path     string
		wantDesc string
	}{
		{"source.filter.regex", "RE2 regex."},
		{"source.filter.any.regex", "RE2 regex."},
		{"source.filter.any.any.regex", "RE2 regex."},
	}
	for _, tc := range cases {
		meta := src.FieldHint("categories", tc.path)
		if meta.Description != tc.wantDesc {
			t.Errorf("path %q: want Description=%q, got %q", tc.path, tc.wantDesc, meta.Description)
		}
	}
}
