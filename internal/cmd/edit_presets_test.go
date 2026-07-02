package cmd

import (
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocPresets_AreValidConfigs guards every whole-document preset: each must
// parse and pass category validation, so a malformed template (e.g. an archive
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
