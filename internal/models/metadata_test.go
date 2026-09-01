package models

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/lucasassuncao/yedit/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metadataProvider is implemented by every config struct that describes itself
// to the `edit` TUI.
type metadataProvider interface {
	Metadata() map[string]*metadata.Node
}

// metadataTypes lists every provider in this package. TestMetadataCoverage
// keeps the list honest, so a new provider cannot quietly go untested.
var metadataTypes = []metadataProvider{
	Config{}, Configuration{}, Logging{}, Watch{}, History{}, Defaults{},
	Category{}, CategorySource{}, CategoryDestination{}, ArchiveConfig{},
	CategoryFilter{}, MatchFilter{}, AgeFilter{}, SizeFilter{},
	CategoryHooks{}, CategoryHook{},
}

// TestMetadataMatchesStructFields guards the one drift that nothing else
// catches: Metadata() is a hand-written map keyed by yaml field name, and the
// `edit` TUI renders exactly what it finds there. A field added to a struct
// without a node is invisible in the editor; a node left behind after a field
// is renamed describes something that no longer exists. Both compile and both
// pass every other test.
func TestMetadataMatchesStructFields(t *testing.T) {
	for _, provider := range metadataTypes {
		typ := reflect.TypeOf(provider)
		t.Run(typ.Name(), func(t *testing.T) {
			fields := yamlFieldNames(typ)
			meta := provider.Metadata()

			for _, field := range fields {
				assert.Contains(t, meta, field, "field %q has no metadata node, so `edit` will not show it", field)
			}
			for name := range meta {
				assert.Contains(t, fields, name, "metadata node %q describes a field that does not exist", name)
			}
		})
	}
}

// TestMetadataCoverage asserts metadataTypes lists every type in the package
// that implements Metadata(), by reading the declarations from the source.
func TestMetadataCoverage(t *testing.T) {
	listed := make(map[string]bool, len(metadataTypes))
	for _, provider := range metadataTypes {
		listed[reflect.TypeOf(provider).Name()] = true
	}

	for _, name := range declaredMetadataReceivers(t) {
		assert.True(t, listed[name], "type %q implements Metadata() but is missing from metadataTypes", name)
	}
}

// yamlFieldNames returns the yaml names of the exported fields of typ, skipping
// the ones marked `yaml:"-"` (computed values that never reach the file).
func yamlFieldNames(typ reflect.Type) []string {
	names := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		names = append(names, name)
	}
	return names
}

// declaredMetadataReceivers returns the receiver type names of every Metadata()
// method declared in this package's source files.
func declaredMetadataReceivers(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	fset := token.NewFileSet()
	var receivers []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Metadata" || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
				receivers = append(receivers, ident.Name)
			}
		}
	}
	require.NotEmpty(t, receivers, "no Metadata() declarations found; the parser is looking at the wrong place")
	return receivers
}
