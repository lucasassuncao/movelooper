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

// testAppBuilder defines the structure for test cases of the AppBuilder chain,
// containing YAML content, a bad path flag, the builder chain to run, an error expectation flag,
// and a check function for assertions on the resulting Movelooper.
type testAppBuilder struct {
	name    string
	yaml    string
	badPath bool
	run     func(m *models.Movelooper, path string) error
	wantErr bool
	check   func(t *testing.T, m *models.Movelooper)
}

// testAppBuilderTestCases defines a set of test cases for the AppBuilder chain,
// covering error propagation, config resolution, logger configuration, config loading,
// category loading, and invalid category validation.
var testAppBuilderTestCases = []testAppBuilder{
	{
		name:    "error stops chain",
		badPath: true,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).
				ResolveConfig().ConfigureLogger().LoadConfig().
				LoadCategories().InitHistory().ValidateDirectories().Build()
		},
		wantErr: true,
		check: func(t *testing.T, m *models.Movelooper) {
			assert.Nil(t, m.Logger)
		},
	},
	{
		name:    "resolve config file not found returns error",
		badPath: true,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).ResolveConfig().Build()
		},
		wantErr: true,
	},
	{
		name: "configure logger sets logger",
		yaml: minimalBuilderYAML,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).ResolveConfig().ConfigureLogger().Build()
		},
		check: func(t *testing.T, m *models.Movelooper) {
			assert.NotNil(t, m.Logger)
		},
	},
	{
		name: "load config sets output",
		yaml: minimalBuilderYAML,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).ResolveConfig().ConfigureLogger().LoadConfig().Build()
		},
		check: func(t *testing.T, m *models.Movelooper) {
			assert.Equal(t, "console", m.Config.Output)
		},
	},
	{
		name: "load categories populates slice",
		yaml: minimalBuilderYAML,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).ResolveConfig().ConfigureLogger().LoadConfig().LoadCategories().Build()
		},
		check: func(t *testing.T, m *models.Movelooper) {
			require.Len(t, m.Categories, 1)
			assert.Equal(t, "docs", m.Categories[0].Name)
		},
	},
	{
		name: "invalid categories returns error with extensions message",
		yaml: `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`,
		run: func(m *models.Movelooper, path string) error {
			return NewAppBuilder(m, path).
				ResolveConfig().ConfigureLogger().LoadConfig().LoadCategories().Build()
		},
		wantErr: true,
		check: func(t *testing.T, m *models.Movelooper) {
		},
	},
	{
		name: "build with no steps returns no error",
		run: func(m *models.Movelooper, path string) error {
			return (&AppBuilder{k: koanf.New(".")}).Build()
		},
	},
}

// TestAppBuilder tests the AppBuilder chain with various configurations
// to ensure it correctly builds a Movelooper or propagates errors.
func TestAppBuilder(t *testing.T) {
	for _, tt := range testAppBuilderTestCases {
		t.Run(tt.name, func(t *testing.T) {
			m := &models.Movelooper{}
			var path string
			if tt.badPath {
				path = "/nonexistent/path/movelooper.yaml"
			} else {
				dir := t.TempDir()
				path = filepath.Join(dir, "cfg.yaml")
				require.NoError(t, os.WriteFile(path, []byte(tt.yaml), 0644))
			}

			err := tt.run(m, path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
	}
}
