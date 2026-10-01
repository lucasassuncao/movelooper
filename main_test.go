package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// withArgs swaps os.Args for the duration of the test, since run reads the
// command line through cobra.
func withArgs(t *testing.T, args ...string) {
	t.Helper()
	saved := os.Args
	os.Args = append([]string{"movelooper"}, args...)
	t.Cleanup(func() { os.Args = saved })
}

func TestRunWiresTheRootCommand(t *testing.T) {
	withArgs(t, "--version")
	require.NoError(t, run())
}

func TestRunReturnsCommandErrors(t *testing.T) {
	withArgs(t, "--no-such-flag")
	require.Error(t, run())
}
