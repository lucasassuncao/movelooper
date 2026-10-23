package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// ImportPath resolves an entry the way the loader does: relative to the file
// that declares it, with ~ expanded, and an absolute one as written.
func TestImportPathResolvesLikeTheLoader(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	main := filepath.Join(dir, "movelooper.yaml")
	other := filepath.Join(t.TempDir(), "abs.yaml")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	cases := map[string]string{
		"./movelooper/wallhaven.yaml": filepath.Join(dir, "movelooper", "wallhaven.yaml"),
		"../shared/a.yaml":            filepath.Join(filepath.Dir(dir), "shared", "a.yaml"),
		other:                         other,
		"~/x.yaml":                    filepath.Join(home, "x.yaml"),
	}
	for imp, want := range cases {
		got, err := ImportPath(main, imp)
		require.NoError(t, err)
		assert.Equal(t, want, got, imp)
	}
}

// testResolveImports defines the structure for test cases of the ResolveImports function,
// containing a setup function, expected error substring, any-error flag, and a check function.
type testResolveImports struct {
	name       string
	setup      func(t *testing.T, dir string) string
	wantErr    string
	wantAnyErr bool
	check      func(t *testing.T, data []byte)
}

// testResolveImportsTestCases defines a set of test cases for the ResolveImports function,
// covering no imports, empty files, single/multiple/nested imports, circular imports, missing files, and malformed YAML.
var testResolveImportsTestCases = []testResolveImports{
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
		name: "diamond import is not circular",
		setup: func(t *testing.T, dir string) string {
			writeYAML(t, dir, "common.yaml", `
categories:
  - name: common
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
			writeYAML(t, dir, "b.yaml", `
import:
  - common.yaml
categories:
  - name: b
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
			writeYAML(t, dir, "c.yaml", `
import:
  - common.yaml
categories:
  - name: c
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
			return writeYAML(t, dir, "main.yaml", `
import:
  - b.yaml
  - c.yaml
categories:
  - name: main
    source: {path: /tmp, extensions: [txt]}
    destination: {path: /tmp}
`)
		},
		check: func(t *testing.T, data []byte) {
			// main + b + common + c; the shared common.yaml is merged once.
			assert.Equal(t, 4, countCategories(t, data))
		},
	},
	{
		name:    "circular import",
		wantErr: "circular import",
		setup: func(t *testing.T, dir string) string {
			aPath := filepath.Join(dir, "a.yaml")
			bPath := filepath.Join(dir, "b.yaml")
			require.NoError(t, os.WriteFile(aPath, []byte("import:\n  - b.yaml\ncategories:\n  - name: a\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0o644))
			require.NoError(t, os.WriteFile(bPath, []byte("import:\n  - a.yaml\ncategories:\n  - name: b\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0o644))
			return aPath
		},
	},
	{
		name:    "sibling circular chain",
		wantErr: "circular import",
		setup: func(t *testing.T, dir string) string {
			bPath := filepath.Join(dir, "b.yaml")
			cPath := filepath.Join(dir, "c.yaml")
			require.NoError(t, os.WriteFile(bPath, []byte("import:\n  - c.yaml\ncategories:\n  - name: b\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0o644))
			require.NoError(t, os.WriteFile(cPath, []byte("import:\n  - b.yaml\ncategories:\n  - name: c\n    source: {path: /tmp, extensions: [txt]}\n    destination: {path: /tmp}\n"), 0o644))
			return writeYAML(t, dir, "main.yaml", "import:\n  - b.yaml\n")
		},
	},
	{
		name: "absolute import is used as written",
		setup: func(t *testing.T, dir string) string {
			// Another directory, the way a file on another drive has no relative form.
			other := writeYAML(t, t.TempDir(), "elsewhere.yaml", `
categories:
  - name: images
    source: {path: /tmp/img, extensions: [jpg]}
    destination: {path: /tmp/dst}
`)
			return writeYAML(t, dir, "main.yaml", "import:\n  - '"+other+"'\n")
		},
		check: func(t *testing.T, data []byte) {
			assert.Equal(t, 1, countCategories(t, data))
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

// TestResolveImports tests the ResolveImports function to ensure it correctly merges
// imported YAML files, detects circular imports, and handles missing or malformed files.
func TestResolveImports(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolveImportsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

// countCategories unmarshals merged YAML bytes and returns the number of categories.
func countCategories(t *testing.T, data []byte) int {
	t.Helper()
	var doc struct {
		Categories []interface{} `yaml:"categories"`
	}
	require.NoError(t, yaml.Unmarshal(data, &doc))
	return len(doc.Categories)
}

// A leading ~ is the home directory, as it is for every other path setting.
func TestResolveImportsExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeYAML(t, home, "shared.yaml", `
categories:
  - name: images
    source: {path: /tmp/img, extensions: [jpg]}
    destination: {path: /tmp/dst}
`)
	path := writeYAML(t, t.TempDir(), "main.yaml", "import:\n  - ~/shared.yaml\n")

	data, err := ResolveImports(path)
	require.NoError(t, err)
	assert.Equal(t, 1, countCategories(t, data))
}

// CheckImports judges each entry on its own, as the editor's validator needs:
// one broken import must not hide whether the others load.
func TestCheckImportsReportsEachEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "ok.yaml", "categories: []\n")
	writeYAML(t, dir, "loop.yaml", "import:\n  - main.yaml\n")
	main := filepath.Join(dir, "main.yaml")
	other := writeYAML(t, t.TempDir(), "abs.yaml", "categories: []\n")

	errs := CheckImports(main, []string{"ok.yaml", "missing.yaml", "loop.yaml", other})
	require.Len(t, errs, 4)
	assert.NoError(t, errs[0])
	assert.ErrorContains(t, errs[1], "missing.yaml")
	assert.ErrorContains(t, errs[2], "circular import")
	assert.NoError(t, errs[3], "an absolute entry loads as written")
}

// "categories:" with no value is null, not a list: the imports were appended to
// that null and lost, so the config ran with no categories and no error.
func TestResolveImportsIntoAnEmptyCategoriesKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "more.yaml"), []byte("categories:\n  - name: imported\n"), 0o600))
	main := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(main, []byte("import:\n  - ./more.yaml\ncategories:\n"), 0o600))

	out, err := ResolveImports(main)
	require.NoError(t, err)

	var doc struct {
		Categories []struct{ Name string } `yaml:"categories"`
	}
	require.NoError(t, yaml.Unmarshal(out, &doc))
	require.Len(t, doc.Categories, 1)
	assert.Equal(t, "imported", doc.Categories[0].Name)
}

// A categories key holding something other than a list is an error, not a
// silent drop of everything imported into it.
func TestResolveImportsRejectsCategoriesThatAreNotAList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "more.yaml"), []byte("categories:\n  - name: imported\n"), 0o600))
	main := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(main, []byte("import:\n  - ./more.yaml\ncategories:\n  name: x\n"), 0o600))

	_, err := ResolveImports(main)
	require.ErrorContains(t, err, "categories")
}

// An imported file whose categories is a map would have its keys and values
// merged as if they were categories.
func TestResolveImportsRejectsAnImportedMap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "more.yaml"), []byte("categories:\n  name: imported\n"), 0o600))
	main := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(main, []byte("import:\n  - ./more.yaml\n"), 0o600))

	_, err := ResolveImports(main)
	require.ErrorContains(t, err, "categories must be a list")
}

// Relative category paths follow the file that declares them, as imports do:
// they used to follow the working directory, which a scheduled task sets to
// somewhere else entirely.
func TestRelativePathsFollowTheirFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "more")
	require.NoError(t, os.MkdirAll(sub, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "extra.yaml"), []byte(
		"categories:\n  - name: imported\n    source:\n      path: in\n      extensions: [txt]\n      exclude-paths: [skip]\n    destination:\n      path: ../out\n"), 0o600))
	main := filepath.Join(dir, "movelooper.yaml")
	require.NoError(t, os.WriteFile(main, []byte(
		"import:\n  - ./more/extra.yaml\ncategories:\n  - name: main\n    source:\n      path: src\n      extensions: [txt]\n    destination:\n      path: ~/dst\n"), 0o600))

	k := koanf.New(".")
	require.NoError(t, InitConfig(k, main))
	cats, err := UnmarshalConfig(k)
	require.NoError(t, err)
	require.Len(t, cats, 2)

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "src"), cats[0].Source.Path)
	assert.Equal(t, filepath.Join(home, "dst"), cats[0].Destination.Path, "~ is still the home folder")
	assert.Equal(t, filepath.Join(sub, "in"), cats[1].Source.Path)
	assert.Equal(t, filepath.Join(dir, "out"), cats[1].Destination.Path)
	assert.Equal(t, []string{filepath.Join(sub, "skip")}, cats[1].Source.ExcludePaths)
}

func TestAnchorPath(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "cfg")
	abs := filepath.Join(t.TempDir(), "x")
	cases := map[string]string{
		"in":     filepath.Join(dir, "in"),
		"../out": filepath.Join(filepath.Dir(dir), "out"),
		abs:      abs,
		"~/x":    "~/x",
		"":       "",
	}
	if runtime.GOOS == "windows" {
		cases[`\rooted`] = `\rooted`
	}
	for in, want := range cases {
		assert.Equal(t, want, AnchorPath(in, dir), in)
	}
}
