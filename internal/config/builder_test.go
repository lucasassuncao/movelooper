package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalBuilderYAML = `
configuration:
  logging:
    output: console
    level: info
categories:
  - name: docs
    source:
      path: /tmp/src
      extensions: [pdf]
    destination:
      path: /tmp/dst
`

var testBuildCases = []struct {
	name    string
	yaml    string
	badPath bool
	opts    []Option
	wantErr bool
	check   func(t *testing.T, m *models.Movelooper)
}{
	{
		name:    "bad config path returns error and nil logger",
		badPath: true,
		opts:    []Option{WithLogger()},
		wantErr: true,
		check: func(t *testing.T, m *models.Movelooper) {
			assert.Nil(t, m.Logger)
		},
	},
	{
		name: "WithLogger sets logger",
		yaml: minimalBuilderYAML,
		opts: []Option{WithLogger()},
		check: func(t *testing.T, m *models.Movelooper) {
			assert.NotNil(t, m.Logger)
		},
	},
	{
		name: "WithConfig loads output setting",
		yaml: minimalBuilderYAML,
		opts: []Option{WithLogger(), WithConfig()},
		check: func(t *testing.T, m *models.Movelooper) {
			assert.Equal(t, "console", m.Config.Logging.Output)
		},
	},
	{
		name: "WithCategories loads categories",
		yaml: minimalBuilderYAML,
		opts: []Option{WithLogger(), WithConfig(), WithCategories()},
		check: func(t *testing.T, m *models.Movelooper) {
			require.Len(t, m.Categories, 1)
			assert.Equal(t, "docs", m.Categories[0].Name)
		},
	},
	{
		name: "invalid category returns error",
		yaml: `
categories:
  - name: broken
    source:
      path: /tmp/src
    destination:
      path: /tmp/dst
`,
		opts:    []Option{WithLogger(), WithConfig(), WithCategories()},
		wantErr: true,
	},
}

func TestNewApp(t *testing.T) {
	t.Parallel()
	for _, tt := range testBuildCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &models.Movelooper{}
			var path string
			if tt.badPath {
				path = "/nonexistent/path/movelooper.yaml"
			} else {
				dir := t.TempDir()
				path = filepath.Join(dir, "cfg.yaml")
				require.NoError(t, os.WriteFile(path, []byte(tt.yaml), 0o644))
			}

			err := NewApp(m, path, tt.opts...)
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

// TestWrapConfigNotFound verifies the not-found guidance points users at the
// current bootstrap command ('movelooper edit'), not the removed 'init'.
func TestWrapConfigNotFound(t *testing.T) {
	t.Parallel()

	t.Run("no path points to edit, not init", func(t *testing.T) {
		t.Parallel()
		err := wrapConfigNotFound("", ErrConfigNotFound)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "movelooper edit")
		assert.NotContains(t, err.Error(), "movelooper init")
		assert.ErrorIs(t, err, ErrConfigNotFound)
	})

	t.Run("explicit path is echoed", func(t *testing.T) {
		t.Parallel()
		err := wrapConfigNotFound("/tmp/x.yaml", ErrConfigNotFound)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "/tmp/x.yaml")
	})

	t.Run("non-notfound error passes through unchanged", func(t *testing.T) {
		t.Parallel()
		orig := errors.New("boom")
		assert.Equal(t, orig, wrapConfigNotFound("", orig))
	})
}
