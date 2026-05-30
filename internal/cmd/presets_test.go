package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPreset_KnownNames(t *testing.T) {
	names := []string{"basic", "images", "music", "video", "books", "archives", "installers", "regex", "full"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			cfg := ConfigPreset(name)
			require.NotNil(t, cfg, "preset %q should not be nil", name)
			assert.NotEmpty(t, cfg.Categories, "preset %q should have at least one category", name)
			assert.NotEmpty(t, cfg.Configuration.LogLevel, "preset %q should have a log level", name)
		})
	}
}

func TestConfigPreset_UnknownName(t *testing.T) {
	cfg := ConfigPreset("nonexistent")
	assert.Nil(t, cfg)
}

func TestListOfConfigPresets_ContainsAll(t *testing.T) {
	list := ListOfConfigPresets()
	expected := []string{"archives", "basic", "books", "full", "images", "installers", "music", "regex", "video"}
	assert.Equal(t, expected, list)
}

func TestListOfConfigPresets_Sorted(t *testing.T) {
	list := ListOfConfigPresets()
	for i := 1; i < len(list); i++ {
		assert.LessOrEqual(t, list[i-1], list[i], "list should be sorted")
	}
}

func TestConfigPreset_RegexHasFilter(t *testing.T) {
	cfg := ConfigPreset("regex")
	require.NotNil(t, cfg)
	require.Len(t, cfg.Categories, 1)
	assert.NotEmpty(t, cfg.Categories[0].Source.Filter.Regex)
}

func TestConfigPreset_FullHasMultipleCategories(t *testing.T) {
	cfg := ConfigPreset("full")
	require.NotNil(t, cfg)
	assert.Greater(t, len(cfg.Categories), 1)
}
