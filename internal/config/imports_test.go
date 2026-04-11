package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// countCategories unmarshals merged YAML bytes and returns the number of categories.
func countCategories(t *testing.T, data []byte) int {
	t.Helper()
	var doc struct {
		Categories []interface{} `yaml:"categories"`
	}
	require.NoError(t, yaml.Unmarshal(data, &doc))
	return len(doc.Categories)
}

func TestResolveImports_NoImport(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "main.yaml", `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`)
	data, err := ResolveImports(path)
	require.NoError(t, err)
	assert.Equal(t, 1, countCategories(t, data))
}

func TestResolveImports_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "empty.yaml", "")
	data, err := ResolveImports(path)
	require.NoError(t, err)
	// Empty file: no categories key expected
	assert.Empty(t, data)
}

func TestResolveImports_WithSingleImport(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "extra.yaml", `
categories:
  - name: images
    source:
      path: /tmp/img
      extensions: [jpg]
    destination:
      path: /tmp/dst
`)

	path := writeYAML(t, dir, "main.yaml", `
import:
  - extra.yaml
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`)

	data, err := ResolveImports(path)
	require.NoError(t, err)
	// Should have docs (from main) + images (from extra)
	assert.Equal(t, 2, countCategories(t, data))
	// import key should be stripped
	assert.NotContains(t, string(data), "import:")
}

func TestResolveImports_ImportWithNoCategoriesInMain(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "extra.yaml", `
categories:
  - name: images
    source:
      path: /tmp/img
      extensions: [jpg]
    destination:
      path: /tmp/dst
`)

	path := writeYAML(t, dir, "main.yaml", `
import:
  - extra.yaml
`)

	data, err := ResolveImports(path)
	require.NoError(t, err)
	assert.Equal(t, 1, countCategories(t, data))
}

func TestResolveImports_CircularImport(t *testing.T) {
	dir := t.TempDir()

	// a.yaml imports b.yaml, b.yaml imports a.yaml
	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")

	require.NoError(t, os.WriteFile(aPath, []byte(`
import:
  - b.yaml
categories:
  - name: a
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`), 0644))
	require.NoError(t, os.WriteFile(bPath, []byte(`
import:
  - a.yaml
categories:
  - name: b
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`), 0644))

	_, err := ResolveImports(aPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular import")
}

func TestResolveImports_MissingImportFile(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "main.yaml", `
import:
  - nonexistent.yaml
`)
	_, err := ResolveImports(path)
	assert.Error(t, err)
}

func TestResolveImports_NestedImports(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "deep.yaml", `
categories:
  - name: deep
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
	writeYAML(t, dir, "mid.yaml", `
import:
  - deep.yaml
categories:
  - name: mid
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
	path := writeYAML(t, dir, "main.yaml", `
import:
  - mid.yaml
categories:
  - name: main
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)

	data, err := ResolveImports(path)
	require.NoError(t, err)
	// main + mid + deep = 3 categories
	assert.Equal(t, 3, countCategories(t, data))
}

func TestResolveImports_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "bad.yaml", "categories: [invalid: yaml: :")
	_, err := ResolveImports(path)
	assert.Error(t, err)
}

func TestResolveImports_MultipleImports(t *testing.T) {
	dir := t.TempDir()

	writeYAML(t, dir, "a.yaml", `
categories:
  - name: alpha
    source: {path: /tmp, extensions: [pdf]}
    destination: {path: /tmp}
`)
	writeYAML(t, dir, "b.yaml", `
categories:
  - name: beta
    source: {path: /tmp, extensions: [jpg]}
    destination: {path: /tmp}
`)
	path := writeYAML(t, dir, "main.yaml", `
import:
  - a.yaml
  - b.yaml
`)

	data, err := ResolveImports(path)
	require.NoError(t, err)
	assert.Equal(t, 2, countCategories(t, data))
}

func TestResolveImports_SiblingCircularChain(t *testing.T) {
	// A imports B; B imports C; C imports B — cycle at the B->C->B level
	dir := t.TempDir()

	bPath := filepath.Join(dir, "b.yaml")
	cPath := filepath.Join(dir, "c.yaml")

	require.NoError(t, os.WriteFile(bPath, []byte(`
import:
  - c.yaml
categories:
  - name: b
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`), 0644))
	require.NoError(t, os.WriteFile(cPath, []byte(`
import:
  - b.yaml
categories:
  - name: c
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`), 0644))

	path := writeYAML(t, dir, "main.yaml", `
import:
  - b.yaml
`)

	_, err := ResolveImports(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular import")
}
