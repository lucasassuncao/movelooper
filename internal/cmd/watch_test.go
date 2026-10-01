package cmd

import (
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchCmd_Flags(t *testing.T) {
	t.Parallel()

	cmd := WatchCmd(&models.Movelooper{})

	for name, def := range map[string]string{
		"show-files":       "false",
		"category":         "",
		"include-disabled": "false",
	} {
		f := cmd.Flags().Lookup(name)
		require.NotNil(t, f, "flag --%s", name)
		assert.Equal(t, def, f.DefValue, "default of --%s", name)
	}

	_, ok := cmd.GetFlagCompletionFunc("category")
	assert.True(t, ok, "--category completes category names")
}
