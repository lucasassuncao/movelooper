package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalBuilderYAML = `
configuration:
  output: console
  log-level: info
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`

func TestAppBuilder_ErrorStopsChain(t *testing.T) {
	m := &models.Movelooper{}
	err := NewAppBuilder(m, "/nonexistent/path/movelooper.yaml").
		ResolveConfig().
		ConfigureLogger().
		LoadConfig().
		LoadCategories().
		InitHistory().
		ValidateDirectories().
		Build()

	require.Error(t, err)
	assert.Nil(t, m.Logger)
}

func TestAppBuilder_ResolveConfig_FileNotFound(t *testing.T) {
	m := &models.Movelooper{}
	b := NewAppBuilder(m, "/nonexistent/path/movelooper.yaml").ResolveConfig()
	assert.Error(t, b.Build())
}

func TestAppBuilder_ConfigureLogger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimalBuilderYAML), 0644))

	m := &models.Movelooper{}
	b := NewAppBuilder(m, path).ResolveConfig().ConfigureLogger()
	require.NoError(t, b.Build())
	assert.NotNil(t, m.Logger)
}

func TestAppBuilder_LoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimalBuilderYAML), 0644))

	m := &models.Movelooper{}
	b := NewAppBuilder(m, path).ResolveConfig().ConfigureLogger().LoadConfig()
	require.NoError(t, b.Build())
	assert.Equal(t, "console", m.Config.Output)
}

func TestAppBuilder_LoadCategories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(path, []byte(minimalBuilderYAML), 0644))

	m := &models.Movelooper{}
	b := NewAppBuilder(m, path).ResolveConfig().ConfigureLogger().LoadConfig().LoadCategories()
	require.NoError(t, b.Build())
	require.Len(t, m.Categories, 1)
	assert.Equal(t, "docs", m.Categories[0].Name)
}

func TestAppBuilder_InvalidCategories_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	yaml := `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0644))

	m := &models.Movelooper{}
	err := NewAppBuilder(m, path).
		ResolveConfig().
		ConfigureLogger().
		LoadConfig().
		LoadCategories().
		Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "extensions")
}

func TestAppBuilder_Build_NilKoanf(t *testing.T) {
	b := &AppBuilder{k: koanf.New(".")}
	assert.NoError(t, b.Build())
}
