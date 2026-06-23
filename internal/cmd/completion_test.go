package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootWithConfigFlag builds a minimal command exposing only the persistent
// --config flag, so the completion func can be unit-tested without RootCmd
// (which mutates package-global command state and is not safe to build
// concurrently from parallel tests).
func rootWithConfigFlag() *cobra.Command {
	c := &cobra.Command{Use: "movelooper"}
	c.PersistentFlags().StringP("config", "c", "", "")
	return c
}

// writeCompletionConfig writes a two-category config and returns its path.
func writeCompletionConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	require.NoError(t, os.MkdirAll(src, 0o750))
	require.NoError(t, os.MkdirAll(dst, 0o750))

	cfg := filepath.Join(dir, "movelooper.yaml")
	yaml := "categories:\n" +
		"  - name: images\n    enabled: true\n    source:\n      path: '" + src + "'\n      extensions: [jpg]\n    destination:\n      path: '" + dst + "'\n" +
		"  - name: docs\n    enabled: true\n    source:\n      path: '" + src + "'\n      extensions: [pdf]\n    destination:\n      path: '" + dst + "'\n"
	require.NoError(t, os.WriteFile(cfg, []byte(yaml), 0o644))
	return cfg
}

// TestCategoryCompletion_EndToEnd drives cobra's real completion machinery
// (the __complete request) to confirm --category suggests category names and
// that the completion path does not require the full app init to succeed.
func TestCategoryCompletion_EndToEnd(t *testing.T) {
	// Not parallel: RootCmd mutates package-global command state.
	cfg := writeCompletionConfig(t)

	root := RootCmd(&models.Movelooper{}, "test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{cobra.ShellCompRequestCmd, "--config", cfg, "--category", ""})
	require.NoError(t, root.Execute())

	out := buf.String()
	assert.Contains(t, out, "images")
	assert.Contains(t, out, "docs")
}

// TestCategoryNameCompletion verifies that --category completion loads the config
// pointed at by --config and returns the configured category names.
func TestCategoryNameCompletion(t *testing.T) {
	t.Parallel()
	cfg := writeCompletionConfig(t)

	root := rootWithConfigFlag()
	require.NoError(t, root.PersistentFlags().Set("config", cfg))

	names, directive := categoryNameCompletion(root, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"images", "docs"}, names)
}

// TestCategoryNameCompletion_NoConfigYieldsNoSuggestions verifies that a missing
// config never breaks completion: it simply returns no suggestions.
func TestCategoryNameCompletion_NoConfigYieldsNoSuggestions(t *testing.T) {
	t.Parallel()
	root := rootWithConfigFlag()
	require.NoError(t, root.PersistentFlags().Set("config", filepath.Join(t.TempDir(), "does-not-exist.yaml")))

	names, directive := categoryNameCompletion(root, nil, "")
	assert.Empty(t, names)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
