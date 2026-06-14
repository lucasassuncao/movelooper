package cmd

import "testing"

// TestBuildMovelooperHints_metadataReachesFieldHint guards the contract the
// FromMetadata validators depend on: every constraint declared in the hint tree
// must resolve through FieldHint, and the strict Build must accept the tree.
func TestBuildMovelooperHints_metadataReachesFieldHint(t *testing.T) {
	t.Parallel()
	src, err := buildMovelooperHints()
	if err != nil {
		t.Fatalf("buildMovelooperHints: %v", err)
	}

	required := []struct{ block, path string }{
		{"configuration", ""},
		{"configuration", "output"},
		{"configuration", "log-level"},
		{"categories", ""},
		{"categories", "name"},
		{"categories", "source"},
		{"categories", "source.path"},
		{"categories", "source.extensions"},
		{"categories", "destination"},
		{"categories", "destination.path"},
		{"categories", "hooks.before.run"},
		{"categories", "hooks.before.on-failure"},
		{"categories", "hooks.after.run"},
		{"categories", "hooks.after.on-failure"},
	}
	for _, f := range required {
		if !src.FieldMeta(f.block, f.path).Required {
			t.Errorf("FieldMeta(%q, %q).Required = false, want true", f.block, f.path)
		}
	}

	notRequired := []struct{ block, path string }{
		{"categories", "enabled"},
		{"categories", "source.filter"},
		{"categories", "hooks"},
		{"categories", "hooks.before"},
	}
	for _, f := range notRequired {
		if src.FieldMeta(f.block, f.path).Required {
			t.Errorf("FieldMeta(%q, %q).Required = true, want false", f.block, f.path)
		}
	}

	oneOf := []struct {
		block, path string
		count       int
	}{
		{"configuration", "output", 4},
		{"configuration", "log-level", 6},
		{"categories", "destination.action", 3},
		{"categories", "destination.conflict-strategy", 8},
		{"categories", "hooks.before.on-failure", 2},
		{"categories", "hooks.after.on-failure", 2},
	}
	for _, f := range oneOf {
		if got := len(src.FieldMeta(f.block, f.path).OneOf); got != f.count {
			t.Errorf("FieldMeta(%q, %q).OneOf has %d values, want %d", f.block, f.path, got, f.count)
		}
	}

	ranged := []struct{ block, path string }{
		{"configuration", "watch-delay"},
		{"configuration", "history-limit"},
		{"categories", "source.max-depth"},
		{"categories", "source.filter.age.min"},
		{"categories", "source.filter.size.max"},
		// the shared filter children must resolve at nested levels too
		{"categories", "source.filter.any.age.min"},
		{"categories", "source.filter.all.any.size.max"},
	}
	for _, f := range ranged {
		meta := src.FieldMeta(f.block, f.path)
		if meta.Min == "" || meta.Max == "" {
			t.Errorf("FieldMeta(%q, %q) should declare Min and Max; got [%q, %q]", f.block, f.path, meta.Min, meta.Max)
		}
	}

	ext := src.FieldMeta("categories", "source.extensions")
	if ext.MinCount != 1 || !ext.Unique {
		t.Errorf("extensions should declare MinCount 1 and Unique; got MinCount=%d Unique=%v", ext.MinCount, ext.Unique)
	}
	if run := src.FieldMeta("categories", "hooks.before.run"); run.MinCount != 1 {
		t.Errorf("hooks.before.run should declare MinCount 1; got %d", run.MinCount)
	}
}
