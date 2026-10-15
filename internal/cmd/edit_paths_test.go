package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
