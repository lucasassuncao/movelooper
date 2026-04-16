package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lucasassuncao/movelooper/internal/models"
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

const minimalCategory = `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`

// --- InitConfig ---

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		nonExistent bool
		wantErr     bool
		errIs       error
	}{
		{"file not found", "", true, true, ErrConfigNotFound},
		{"malformed yaml", "categories: [invalid: yaml: :", false, true, nil},
		{"valid minimal config", minimalCategory, false, false, nil},
		{"empty file is valid", "", false, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var path string
			if tt.nonExistent {
				path = "/nonexistent/path/movelooper.yaml"
			} else {
				path = writeYAML(t, dir, "cfg.yaml", tt.yaml)
			}

			k := koanf.New(".")
			err := InitConfig(k, path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

// --- UnmarshalConfig ---

func TestUnmarshalConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string // substring; empty = no error expected
		check   func(t *testing.T, cats []*models.Category)
	}{
		{
			name: "valid category",
			yaml: `
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf, txt]
    destination:
      path: /tmp/dst
`,
			check: func(t *testing.T, cats []*models.Category) {
				require.Len(t, cats, 1)
				assert.Equal(t, "docs", cats[0].Name)
				assert.ElementsMatch(t, []string{"pdf", "txt"}, cats[0].Source.Extensions)
			},
		},
		{
			name:    "missing extensions",
			wantErr: "source.extensions are required",
			yaml: `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`,
		},
		{
			name:    "invalid regex",
			wantErr: "invalid regex",
			yaml: `
categories:
  - name: bad-regex
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        regex: "[invalid"
    destination:
      path: /tmp/dst
`,
		},
		{
			name:    "regex and glob mutually exclusive",
			wantErr: "mutually exclusive",
			yaml: `
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
`,
		},
		{
			name:    "min-size greater than max-size",
			wantErr: "min-size",
			yaml: `
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
`,
		},
		{
			name:    "min-age greater than max-age",
			wantErr: "min-age",
			yaml: `
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
`,
		},
		{
			name: "case-insensitive regex compiled",
			yaml: `
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
`,
			check: func(t *testing.T, cats []*models.Category) {
				require.NotNil(t, cats[0].Source.Filter.CompiledRegex)
				assert.True(t, cats[0].Source.Filter.CompiledRegex.MatchString("REPORT"))
			},
		},
		{
			name: "case-sensitive regex compiled",
			yaml: `
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
`,
			check: func(t *testing.T, cats []*models.Category) {
				require.NotNil(t, cats[0].Source.Filter.CompiledRegex)
				assert.False(t, cats[0].Source.Filter.CompiledRegex.MatchString("REPORT"))
				assert.True(t, cats[0].Source.Filter.CompiledRegex.MatchString("report"))
			},
		},
		{
			name: "size bytes populated",
			yaml: `
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
`,
			check: func(t *testing.T, cats []*models.Category) {
				assert.Equal(t, int64(1024), cats[0].Source.Filter.MinSizeBytes)
				assert.Equal(t, int64(10*1024*1024), cats[0].Source.Filter.MaxSizeBytes)
			},
		},
		{
			name:    "invalid glob",
			wantErr: "",
			yaml: `
categories:
  - name: bad-glob
    source:
      path: /tmp/src
      extensions: [txt]
      filter:
        glob: "[invalid"
    destination:
      path: /tmp/dst
`,
			check: func(t *testing.T, cats []*models.Category) {
				// error is expected — this case is handled below
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeYAML(t, dir, "cfg.yaml", tt.yaml)
			k := koanf.New(".")
			require.NoError(t, InitConfig(k, path))

			cats, err := UnmarshalConfig(k)

			if tt.name == "invalid glob" {
				assert.Error(t, err)
				return
			}
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cats)
			}
		})
	}
}

// --- LoadConfig ---

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		check func(t *testing.T, cfg models.Configuration)
	}{
		{
			name: "defaults when not set",
			yaml: "",
			check: func(t *testing.T, cfg models.Configuration) {
				assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
				assert.Equal(t, defaultHistoryLimit, cfg.HistoryLimit)
			},
		},
		{
			name: "custom values",
			yaml: `
configuration:
  output: json
  log-level: debug
  watch-delay: 2m
  history-limit: 100
`,
			check: func(t *testing.T, cfg models.Configuration) {
				assert.Equal(t, "json", cfg.Output)
				assert.Equal(t, "debug", cfg.LogLevel)
				assert.Equal(t, 2*time.Minute, cfg.WatchDelay)
				assert.Equal(t, 100, cfg.HistoryLimit)
			},
		},
		{
			name: "watch-delay fallback to default",
			yaml: "configuration:\n  output: text\n",
			check: func(t *testing.T, cfg models.Configuration) {
				assert.Equal(t, defaultWatchDelay, cfg.WatchDelay)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeYAML(t, dir, "cfg.yaml", tt.yaml)
			k := koanf.New(".")
			if tt.yaml != "" {
				require.NoError(t, InitConfig(k, path))
			}
			tt.check(t, LoadConfig(k))
		})
	}
}

func TestValidateCategory_Action(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{"empty defaults to move — ok", "", false},
		{"move explicit — ok", "move", false},
		{"copy — ok", "copy", false},
		{"symlink — ok", "symlink", false},
		{"invalid action", "link", true},
		{"uppercase invalid", "MOVE", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &models.Category{
				Name: "test",
				Source: models.CategorySource{
					Extensions: []string{"pdf"},
				},
				Destination: models.CategoryDestination{
					Path:   "/tmp/dst",
					Action: tt.action,
				},
			}
			err := validateCategory(cat)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "action")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCategory_Rename(t *testing.T) {
	tests := []struct {
		name    string
		rename  string
		wantErr bool
	}{
		{"empty — ok", "", false},
		{"valid template — ok", "{mod-date}_{name}.{ext}", false},
		{"unknown token — error", "{unknown}", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &models.Category{
				Name: "test",
				Source: models.CategorySource{
					Extensions: []string{"pdf"},
				},
				Destination: models.CategoryDestination{
					Path:   "/tmp/dst",
					Rename: tt.rename,
				},
			}
			err := validateCategory(cat)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "rename")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
