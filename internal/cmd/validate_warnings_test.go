package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// warningPaths returns the config paths flagged by configWarnings, which is
// what the assertions below care about.
func warningPaths(t *testing.T, yaml string) []string {
	t.Helper()
	warnings := configWarnings([]byte(yaml))
	paths := make([]string, 0, len(warnings))
	for _, w := range warnings {
		require.NotEmpty(t, w.Message, "every warning must explain itself")
		paths = append(paths, w.Path)
	}
	return paths
}

// TestConfigWarnings_CollapsingRename covers the worst case: a rename template
// with no per-file token plus overwrite, where every file destroys the previous
// one and a single file survives.
func TestConfigWarnings_CollapsingRename(t *testing.T) {
	t.Parallel()

	t.Run("flags a constant rename with overwrite", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: photos
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/out
      rename: "{category}.{ext}"
      conflict-strategy: overwrite
`)
		assert.Contains(t, paths, "categories[0].destination.rename")
	})

	t.Run("accepts a rename carrying a per-file token", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: photos
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/out
      rename: "{name}-{seq}.{ext}"
      conflict-strategy: overwrite
`)
		assert.NotContains(t, paths, "categories[0].destination.rename")
	})

	t.Run("stays quiet when the strategy keeps earlier files", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: photos
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/out
      rename: "{category}.{ext}"
      conflict-strategy: rename
`)
		assert.Empty(t, paths)
	})

	// The dangerous strategy can arrive from configuration.defaults instead of
	// the category, and the warning must still fire.
	t.Run("resolves overwrite inherited from defaults", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
configuration:
  defaults:
    conflict-strategy: overwrite
categories:
  - name: photos
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/out
      rename: "{category}.{ext}"
`)
		assert.Contains(t, paths, "categories[0].destination.rename")
	})
}

func TestConfigWarnings_SourceEqualsDestination(t *testing.T) {
	t.Parallel()
	paths := warningPaths(t, `
categories:
  - name: photos
    source:
      path: /tmp/same
      extensions: [jpg]
    destination:
      path: /tmp/same/
`)
	assert.Contains(t, paths, "categories[0].destination.path")
}

func TestConfigWarnings_ArchiveDeletingSources(t *testing.T) {
	t.Parallel()

	t.Run("flags keep-source false", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: backup
    source:
      path: /tmp/in
      extensions: [all]
    destination:
      path: /tmp/out
      action: archive
      archive:
        format: zip
        keep-source: false
`)
		assert.Contains(t, paths, "categories[0].destination.archive.keep-source")
	})

	t.Run("stays quiet when sources are kept", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: backup
    source:
      path: /tmp/in
      extensions: [all]
    destination:
      path: /tmp/out
      action: archive
      archive:
        format: zip
        keep-source: true
`)
		assert.Empty(t, paths)
	})
}

func TestConfigWarnings_OverwriteWithoutSeparation(t *testing.T) {
	t.Parallel()

	t.Run("flags overwrite with no organize-by and no rename", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: docs
    source:
      path: /tmp/in
      extensions: [pdf]
    destination:
      path: /tmp/out
      conflict-strategy: overwrite
`)
		assert.Contains(t, paths, "categories[0].destination.conflict-strategy")
	})

	t.Run("stays quiet when organize-by separates the files", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: docs
    source:
      path: /tmp/in
      extensions: [pdf]
    destination:
      path: /tmp/out
      organize-by: "{mod-year}"
      conflict-strategy: overwrite
`)
		assert.Empty(t, paths)
	})
}

func TestConfigWarnings_CompetingCategories(t *testing.T) {
	t.Parallel()

	t.Run("flags two categories sharing a source and an extension", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: first
    source:
      path: /tmp/in
      extensions: [jpg, png]
    destination:
      path: /tmp/a
  - name: second
    source:
      path: /tmp/in
      extensions: [png, gif]
    destination:
      path: /tmp/b
`)
		assert.Contains(t, paths, "categories[1].source")
	})

	t.Run("flags a catch-all competing with a specific category", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: everything
    source:
      path: /tmp/in
      extensions: [all]
    destination:
      path: /tmp/a
  - name: photos
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/b
`)
		assert.Contains(t, paths, "categories[1].source")
	})

	t.Run("stays quiet for different sources", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: first
    source:
      path: /tmp/one
      extensions: [jpg]
    destination:
      path: /tmp/a
  - name: second
    source:
      path: /tmp/two
      extensions: [jpg]
    destination:
      path: /tmp/b
`)
		assert.Empty(t, paths)
	})

	t.Run("stays quiet for the same source with disjoint extensions", func(t *testing.T) {
		t.Parallel()
		paths := warningPaths(t, `
categories:
  - name: first
    source:
      path: /tmp/in
      extensions: [jpg]
    destination:
      path: /tmp/a
  - name: second
    source:
      path: /tmp/in
      extensions: [pdf]
    destination:
      path: /tmp/b
`)
		assert.Empty(t, paths)
	})
}

// TestConfigWarnings_UnparseableConfig makes sure warnings never get in the way
// of the real validation errors when the file does not even parse.
func TestConfigWarnings_UnparseableConfig(t *testing.T) {
	t.Parallel()
	assert.Nil(t, configWarnings([]byte("categories: [oops")))
	assert.Nil(t, configWarnings(nil))
}

// TestConfigWarnings_CleanConfigIsSilent guards against warning noise on a
// perfectly ordinary configuration.
func TestConfigWarnings_CleanConfigIsSilent(t *testing.T) {
	t.Parallel()
	paths := warningPaths(t, `
configuration:
  logging:
    output: console
    level: info
categories:
  - name: images
    enabled: true
    source:
      path: /tmp/downloads
      extensions: [jpg, png]
    destination:
      path: /tmp/pictures
      organize-by: "{mod-year}"
      conflict-strategy: rename
`)
	assert.Empty(t, paths)
}
