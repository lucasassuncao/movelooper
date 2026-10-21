package cmd

import (
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/presets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "go.yaml.in/yaml/v3"
)

// TestDocPresets_AreValidConfigs guards every whole-document preset: each must
// parse and pass category validation, so a malformed preset (e.g. an archive
// preset missing its archive block) is caught here rather than by the user.
func TestDocPresets_AreValidConfigs(t *testing.T) {
	t.Parallel()
	names := MovelooperDocPresets.ListPresets("")
	require.NotEmpty(t, names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			y, err := MovelooperDocPresets.PresetYAML("", name)
			require.NoError(t, err)

			k := koanf.New(".")
			require.NoError(t, k.Load(rawbytes.Provider([]byte(y)), yaml.Parser()))

			_, err = config.UnmarshalConfig(k)
			assert.NoErrorf(t, err, "doc preset %q must be a valid config", name)
		})
	}
}

// TestPresets_IndentWithTwoSpaces guards the editor's style: presets land in
// the file as they are, so they must use the 2 spaces yedit writes edits with.
func TestPresets_IndentWithTwoSpaces(t *testing.T) {
	t.Parallel()
	sources := map[string]presets.Source{"doc": MovelooperDocPresets, "block": MovelooperBlockPresets}
	for kind, src := range sources {
		for _, field := range src.ListFields() {
			for _, name := range src.ListPresets(field) {
				y, err := src.PresetYAML(field, name)
				require.NoError(t, err)
				second := strings.SplitN(y, "\n", 3)[1]
				assert.Regexpf(t, `^  \S`, second, "%s preset %q:\n%s", kind, name, y)
			}
		}
	}
}

// TestCategoryPresets_AreValidConfigs guards every category (block) preset: each
// must pass category validation when loaded as a config.
func TestCategoryPresets_AreValidConfigs(t *testing.T) {
	t.Parallel()
	names := ListOfCategoriesPresets()
	require.NotEmpty(t, names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cats := CategoriesPreset(name)
			require.NotEmpty(t, cats)

			raw, err := yamlv3.Marshal(models.Config{Categories: cats})
			require.NoError(t, err)

			k := koanf.New(".")
			require.NoError(t, k.Load(rawbytes.Provider(raw), yaml.Parser()))

			_, err = config.UnmarshalConfig(k)
			assert.NoErrorf(t, err, "category preset %q must be a valid config", name)
		})
	}
}
