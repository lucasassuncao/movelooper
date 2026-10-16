package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/spec"
	"go.yaml.in/yaml/v3"
)

// editPathFields makes import a list of paths in the editor: ctrl+o picks a
// config file and writes it the way the rest of the list is written.
var editPathFields = map[string]editor.PathField{
	"import": {Types: []string{".yaml", ".yml"}, Format: importPath},
}

// importPath spells a picked import like movelooper.yaml does: relative to the
// config's folder, "./" first and forward slashes, on every platform.
func importPath(abs, docPath string) (string, error) {
	rel, err := filepath.Rel(filepath.Dir(docPath), abs)
	if err != nil {
		// Another drive has no relative form; the loader reads an absolute one.
		return abs, nil
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "../") {
		rel = "./" + rel
	}
	return rel, nil
}

// importsResolve flags each import entry that would not load from docPath,
// where the config is saved: with --output elsewhere, relative ones can break.
func importsResolve(docPath string) spec.Validator {
	return spec.ValidatorFunc(func(in spec.ValidationInput) []spec.Violation {
		var doc struct {
			Import []string `yaml:"import"`
		}
		if err := yaml.Unmarshal(in.Raw, &doc); err != nil {
			return nil
		}
		var out []spec.Violation
		for i, err := range config.CheckImports(docPath, doc.Import) {
			if err != nil {
				out = append(out, spec.Violation{Path: fmt.Sprintf("import[%d]", i), Message: err.Error()})
			}
		}
		return out
	})
}
