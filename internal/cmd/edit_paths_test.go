package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Picks are written the way movelooper.yaml spells imports: relative to the
// config's folder, with "./" and forward slashes on every platform.
func TestImportPathFollowsTheConfigConvention(t *testing.T) {
	t.Parallel()
	conf := filepath.Join(t.TempDir(), "conf")
	doc := filepath.Join(conf, "movelooper.yaml")
	cases := map[string]string{
		filepath.Join(conf, "movelooper", "wallhaven.yaml"):      "./movelooper/wallhaven.yaml",
		filepath.Join(conf, "extra.yaml"):                        "./extra.yaml",
		filepath.Join(filepath.Dir(conf), "shared", "docs.yaml"): "../shared/docs.yaml",
	}
	for abs, want := range cases {
		got, err := importPath(abs, doc)
		require.NoError(t, err)
		assert.Equal(t, want, got, abs)
	}
}

// A file on another drive has no relative form: it is written absolute, which
// the loader reads as written.
func TestImportPathKeepsAnotherDriveAbsolute(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("drive letters are a Windows matter")
	}
	got, err := importPath(`E:\shared\docs.yaml`, `D:\Scripts\go\conf\movelooper.yaml`)
	require.NoError(t, err)
	assert.Equal(t, `E:\shared\docs.yaml`, got)
}

func TestImportsResolveFlagsEntriesThatWouldNotLoad(t *testing.T) {
	t.Parallel()
	conf := t.TempDir()
	doc := filepath.Join(conf, "movelooper.yaml")
	require.NoError(t, os.WriteFile(filepath.Join(conf, "ok.yaml"), []byte("categories: []\n"), 0o600))

	raw := []byte("import:\n  - ./ok.yaml\n  - ./missing.yaml\n")
	got := importsResolve(doc).Validate(spec.NewValidationInput(raw, nil))
	require.Len(t, got, 1)
	assert.Equal(t, "import[1]", got[0].Path)
	assert.Contains(t, got[0].Message, "missing.yaml")
}

// --output elsewhere: the entries must load from where the file will live.
func TestImportsResolveChecksFromTheSaveTarget(t *testing.T) {
	t.Parallel()
	conf := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(conf, "ok.yaml"), []byte("categories: []\n"), 0o600))
	elsewhere := filepath.Join(t.TempDir(), "copy.yaml")

	raw := []byte("import:\n  - ./ok.yaml\n")
	assert.Empty(t, importsResolve(filepath.Join(conf, "movelooper.yaml")).Validate(spec.NewValidationInput(raw, nil)))
	assert.Len(t, importsResolve(elsewhere).Validate(spec.NewValidationInput(raw, nil)), 1)
}

// An import entry opens its file with the import schema: there the
// configuration block the main config requires is not expected, and the
// file's own imports open in turn.
func TestImportEntryOpensWithTheImportSchema(t *testing.T) {
	t.Parallel()
	conf := t.TempDir()
	doc := filepath.Join(conf, "movelooper.yaml")
	wall := filepath.Join(conf, "movelooper", "wallhaven.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(wall), 0o750))
	require.NoError(t, os.WriteFile(wall, []byte(`categories:
  - name: wallhaven
    enabled: true
    source:
      path: a
      extensions: [jpg]
    destination:
      path: b
`), 0o600))
	hints, err := buildMovelooperHints()
	require.NoError(t, err)
	base := editor.Config{Path: doc, SavePath: filepath.Join(conf, "out.yaml"), Schema: &models.Config{}, Metadata: hints, Validators: MovelooperValidators}

	cfg, err := importFields(base)["import"].Open("./movelooper/wallhaven.yaml", doc)

	require.NoError(t, err)
	assert.Equal(t, wall, cfg.Path)
	assert.Empty(t, cfg.SavePath, "the file is saved where it is")
	assert.NotNil(t, cfg.PathFields["import"].Open, "its own imports open too")
	d, err := document.Load(wall, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, editor.RunAll(editor.Wire(base.Validators, base), d.Raw(), d.Blocks()), "the main schema requires configuration")
	assert.Empty(t, editor.RunAll(editor.Wire(cfg.Validators, cfg), d.Raw(), d.Blocks()))
}
