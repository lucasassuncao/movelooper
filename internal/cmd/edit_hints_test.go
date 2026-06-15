package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildMovelooperHints_metadataReachesFieldHint guards the contract the
// FromMetadata validators depend on: every constraint declared in the hint tree
// must resolve through FieldHint, and the strict Build must accept the tree.
func TestBuildMovelooperHints_metadataReachesFieldHint(t *testing.T) {
	t.Parallel()
	src, err := buildMovelooperHints()
	require.NoError(t, err)

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
		assert.True(t, src.FieldMeta(f.block, f.path).Required, "FieldMeta(%q, %q).Required", f.block, f.path)
	}

	notRequired := []struct{ block, path string }{
		{"categories", "enabled"},
		{"categories", "source.filter"},
		{"categories", "hooks"},
		{"categories", "hooks.before"},
	}
	for _, f := range notRequired {
		assert.False(t, src.FieldMeta(f.block, f.path).Required, "FieldMeta(%q, %q).Required", f.block, f.path)
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
		assert.Len(t, src.FieldMeta(f.block, f.path).OneOf, f.count, "FieldMeta(%q, %q).OneOf", f.block, f.path)
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
		assert.NotEmpty(t, meta.Min, "FieldMeta(%q, %q).Min", f.block, f.path)
		assert.NotEmpty(t, meta.Max, "FieldMeta(%q, %q).Max", f.block, f.path)
	}

	ext := src.FieldMeta("categories", "source.extensions")
	assert.Equal(t, 1, ext.MinCount, "extensions.MinCount")
	assert.True(t, ext.Unique, "extensions.Unique")
	assert.Equal(t, 1, src.FieldMeta("categories", "hooks.before.run").MinCount, "hooks.before.run.MinCount")
}
