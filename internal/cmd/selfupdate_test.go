package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Both paths must fail before reaching the network when no repo is known.
func TestSelfUpdateCmd_NoRepoFails(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"--repo", ""}, {"--repo", "", "--list"}} {
		cmd := SelfUpdateCmd("v1.0.0")
		cmd.SetArgs(args)

		err := cmd.Execute()
		require.Error(t, err, "args %v", args)
		assert.Contains(t, err.Error(), "no repository configured")
	}
}
