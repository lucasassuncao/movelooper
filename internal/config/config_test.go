package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeYAML writes content to a temp file and returns its path.
func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// --- InitConfig ---

func TestInitConfig_FileNotFound(t *testing.T) {
	k := koanf.New(".")
	err := InitConfig(k, "/nonexistent/path/movelooper.yaml")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestInitConfig_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "bad.yaml", "categories: [invalid: yaml: :")
	k := koanf.New(".")
	err := InitConfig(k, path)
	assert.Error(t, err)
}

func TestInitConfig_ValidMinimalConfig(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "movelooper.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))
}

func TestInitConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "empty.yaml", "")
	k := koanf.New(".")
	// Empty file is valid YAML (no content), should not error
	assert.NoError(t, InitConfig(k, path))
}

// --- UnmarshalConfig ---

func TestUnmarshalConfig_ValidCategory(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf, txt]
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cats, err := UnmarshalConfig(k)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, "docs", cats[0].Name)
	assert.ElementsMatch(t, []string{"pdf", "txt"}, cats[0].Source.Extensions)
}

func TestUnmarshalConfig_MissingExtensions(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source.extensions are required")
}

func TestUnmarshalConfig_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: bad-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        regex: "[invalid"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex")
}

func TestUnmarshalConfig_RegexAndGlobMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: both-filters
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        regex: ".*"
        glob: "*.txt"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestUnmarshalConfig_MinSizeGreaterThanMaxSize(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: bad-size
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        min-size: "10 MB"
        max-size: "1 MB"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min-size")
}

func TestUnmarshalConfig_MinAgeGreaterThanMaxAge(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: bad-age
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        min-age: "48h"
        max-age: "24h"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min-age")
}

func TestUnmarshalConfig_CaseInsensitiveRegexCompiled(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: ci-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        regex: "report"
        case-sensitive: false
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cats, err := UnmarshalConfig(k)
	require.NoError(t, err)
	require.NotNil(t, cats[0].Source.Filter.CompiledRegex)
	// (?i) prefix makes it case-insensitive
	assert.True(t, cats[0].Source.Filter.CompiledRegex.MatchString("REPORT"))
}

func TestUnmarshalConfig_CaseSensitiveRegexCompiled(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: cs-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        regex: "report"
        case-sensitive: true
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cats, err := UnmarshalConfig(k)
	require.NoError(t, err)
	require.NotNil(t, cats[0].Source.Filter.CompiledRegex)
	assert.False(t, cats[0].Source.Filter.CompiledRegex.MatchString("REPORT"))
	assert.True(t, cats[0].Source.Filter.CompiledRegex.MatchString("report"))
}

func TestUnmarshalConfig_SizeBytesPopulated(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: sized
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        min-size: "1 KB"
        max-size: "10 MB"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cats, err := UnmarshalConfig(k)
	require.NoError(t, err)
	assert.Equal(t, int64(1024), cats[0].Source.Filter.MinSizeBytes)
	assert.Equal(t, int64(10*1024*1024), cats[0].Source.Filter.MaxSizeBytes)
}

func TestUnmarshalConfig_InvalidGlob(t *testing.T) {
	dir := t.TempDir()
	yaml := `
categories:
  - name: bad-glob
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        glob: "[invalid"
    destination:
      path: /tmp/dst
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	_, err := UnmarshalConfig(k)
	assert.Error(t, err)
}

// --- LoadConfig defaults ---

func TestLoadConfig_Defaults(t *testing.T) {
	k := koanf.New(".")
	cfg := LoadConfig(k)
	assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
	assert.Equal(t, defaultHistoryLimit, cfg.HistoryLimit)
}

func TestLoadConfig_CustomValues(t *testing.T) {
	dir := t.TempDir()
	yaml := `
configuration:
  output: json
  log-level: debug
  watch-delay: 2m
  history-limit: 100
`
	path := writeYAML(t, dir, "cfg.yaml", yaml)
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cfg := LoadConfig(k)
	assert.Equal(t, "json", cfg.Output)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 2*time.Minute, cfg.WatchDelay)
	assert.Equal(t, 100, cfg.HistoryLimit)
}

func TestLoadConfig_WatchDelayFallback(t *testing.T) {
	// When watch-delay is not set, default is used
	dir := t.TempDir()
	path := writeYAML(t, dir, "cfg.yaml", "configuration:\n  output: text\n")
	k := koanf.New(".")
	require.NoError(t, InitConfig(k, path))

	cfg := LoadConfig(k)
	assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
}
