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

func TestResolveImports(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string) string // returns path to main yaml
		wantErr    string                                // expected substring in error message
		wantAnyErr bool                                  // true when any error is expected (without message check)
		check      func(t *testing.T, data []byte)
	}{
		{
			name: "no import",
			setup: func(t *testing.T, dir string) string {
				return writeYAML(t, dir, "main.yaml", `
categories:
  - name: docs
    source: {path: /tmp/src, extensions: [pdf]}
    destination: {path: /tmp/dst}
`)
			},
			check: func(t *testing.T, data []byte) {
				assert.Equal(t, 1, countCategories(t, data))
			},
		},
		{
			name: "empty file",
			setup: func(t *testing.T, dir string) string {
				return writeYAML(t, dir, "empty.yaml", "")
			},
			check: func(t *testing.T, data []byte) {
				assert.Empty(t, data)
			},
		},
		{
			name: "single import merges categories",
			setup: func(t *testing.T, dir string) string {
				writeYAML(t, dir, "extra.yaml", `
categories:
  - name: images
    source: {path: /tmp/img, extensions: [jpg]}
    destination: {path: /tmp/dst}
`)
				return writeYAML(t, dir, "main.yaml", `
import:
  - extra.yaml
categories:
  - name: docs
    source: {path: /tmp/src, extensions: [pdf]}
    destination: {path: /tmp/dst}
`)
			},
			check: func(t *testing.T, data []byte) {
				assert.Equal(t, 2, countCategories(t, data))
				assert.NotContains(t, string(data), "import:")
			},
		},
		{
			name: "import with no categories in main",
			setup: func(t *testing.T, dir string) string {
				writeYAML(t, dir, "extra.yaml", `
categories:
  - name: images
    source: {path: /tmp/img, extensions: [jpg]}
    destination: {path: /tmp/dst}
`)
				return writeYAML(t, dir, "main.yaml", "import:\n  - extra.yaml\n")
			},
			check: func(t *testing.T, data []byte) {
				assert.Equal(t, 1, countCategories(t, data))
			},
		},
		{
			name: "multiple imports",
			setup: func(t *testing.T, dir string) string {
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
				return writeYAML(t, dir, "main.yaml", "import:\n  - a.yaml\n  - b.yaml\n")
			},
			check: func(t *testing.T, data []byte) {
				assert.Equal(t, 2, countCategories(t, data))
			},
		},
		{
			name: "nested imports",
			setup: func(t *testing.T, dir string) string {
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
				return writeYAML(t, dir, "main.yaml", `
import:
  - mid.yaml
categories:
  - name: main
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
			},
			check: func(t *testing.T, data []byte) {
				assert.Equal(t, 3, countCategories(t, data))
			},
		},
		{
			name:    "circular import",
			wantErr: "circular import",
			setup: func(t *testing.T, dir string) string {
				aPath := filepath.Join(dir, "a.yaml")
				bPath := filepath.Join(dir, "b.yaml")
				require.NoError(t, os.WriteFile(aPath, []byte("import:\n  - b.yaml\ncategories:\n  - name: a\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0644))
				require.NoError(t, os.WriteFile(bPath, []byte("import:\n  - a.yaml\ncategories:\n  - name: b\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0644))
				return aPath
			},
		},
		{
			name:    "sibling circular chain",
			wantErr: "circular import",
			setup: func(t *testing.T, dir string) string {
				bPath := filepath.Join(dir, "b.yaml")
				cPath := filepath.Join(dir, "c.yaml")
				require.NoError(t, os.WriteFile(bPath, []byte("import:\n  - c.yaml\ncategories:\n  - name: b\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0644))
				require.NoError(t, os.WriteFile(cPath, []byte("import:\n  - b.yaml\ncategories:\n  - name: c\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0644))
				return writeYAML(t, dir, "main.yaml", "import:\n  - b.yaml\n")
			},
		},
		{
			name:       "missing import file",
			wantAnyErr: true,
			setup: func(t *testing.T, dir string) string {
				return writeYAML(t, dir, "main.yaml", "import:\n  - nonexistent.yaml\n")
			},
		},
		{
			name:       "malformed yaml",
			wantAnyErr: true,
			setup: func(t *testing.T, dir string) string {
				return writeYAML(t, dir, "bad.yaml", "categories: [invalid: yaml: :")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t, t.TempDir())
			data, err := ResolveImports(path)

			switch {
			case tt.wantAnyErr:
				assert.Error(t, err)
			case tt.wantErr != "":
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			default:
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, data)
				}
			}
		})
	}
}
