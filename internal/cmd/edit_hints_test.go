package cmd

import "testing"

// TestBuildMovelooperHints is the guard for the Metadata() map: an unknown key
// or a field with no entry only surfaces here, when the tree is decoded and
// checked against the structs.
func TestBuildMovelooperHints(t *testing.T) {
	src, err := buildMovelooperHints()
	if err != nil {
		t.Fatalf("buildMovelooperHints: %v", err)
	}
	if got := src.FieldMeta("categories", "source.path"); got.Description == "" || !got.Required {
		t.Errorf("categories.source.path = %+v", got)
	}
	if got := src.FieldMeta("configuration", "logging.max-width"); got.Max != "500" {
		t.Errorf("configuration.logging.max-width max = %q, want 500", got.Max)
	}
	if got := src.FieldMeta("categories", "source.filter.any.match.glob"); got.Description == "" {
		t.Error("recursive filter should carry its own metadata at depth")
	}
}
