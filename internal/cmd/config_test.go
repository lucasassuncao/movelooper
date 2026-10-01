package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCmd_MissingConfigFlagFails(t *testing.T) {
	t.Parallel()

	root := rootWithConfigFlag()
	root.AddCommand(ConfigCmd())
	root.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "absent.yaml"), "config"})

	err := root.Execute()
	assert.ErrorIs(t, err, config.ErrConfigNotFound)
}

func TestConfigCmd_ExistingConfigFlagSucceeds(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "movelooper.yaml")
	require.NoError(t, os.WriteFile(path, []byte("categories: []\n"), 0o600))

	root := rootWithConfigFlag()
	root.AddCommand(ConfigCmd())
	root.SetArgs([]string{"--config", path, "config"})

	assert.NoError(t, root.Execute())
}
