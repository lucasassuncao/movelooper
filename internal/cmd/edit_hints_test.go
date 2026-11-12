package cmd

import (
	"regexp"
	"testing"
)

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
	width := regexp.MustCompile(src.FieldMeta("configuration", "logging.max-width").Pattern)
	for v, want := range map[string]bool{"auto": true, "0": true, "7": true, "120": true, "500": true, "501": false, "-1": false, "010": false, "Auto": false, "": false} {
		if width.MatchString(v) != want {
			t.Errorf("configuration.logging.max-width matches %q = %v, want %v", v, !want, want)
		}
	}
	if got := src.FieldMeta("categories", "source.filter.any.match.glob"); got.Description == "" {
		t.Error("recursive filter should carry its own metadata at depth")
	}
}
