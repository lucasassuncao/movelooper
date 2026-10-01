package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSelfUpdateList_RequiresRepo(t *testing.T) {
	t.Parallel()

	err := runSelfUpdateList("", false, 20, "v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repo is required")
}
