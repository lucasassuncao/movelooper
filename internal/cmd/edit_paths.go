package cmd

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/metadata"
	"github.com/lucasassuncao/yedit/spec"
	"go.yaml.in/yaml/v3"
)

// importFields makes import a list of paths in the editor: ctrl+o picks a
// config file and writes it the way the rest of the list is written, and Enter
// on an entry edits that file in a tab of its own, set up like base.
func importFields(base editor.Config) map[string]editor.PathField {
	return map[string]editor.PathField{"import": {
		Types:  []string{".yaml", ".yml"},
		Format: importPath,
		Open: func(value, docPath string) (editor.Config, error) {
			return importConfig(base, value, docPath)
		},
	}}
}

// importConfig is how the file an import entry names is edited: base's look
// and rules, with the import schema, which has no configuration block.
func importConfig(base editor.Config, value, docPath string) (editor.Config, error) {
	abs, err := config.ImportPath(docPath, value)
	if err != nil {
		return editor.Config{}, err
	}
	hints, err := metadata.New(models.ImportFile{})
	if err != nil {
		return editor.Config{}, fmt.Errorf("building hint source: %w", err)
	}
	cfg := base
	cfg.Path, cfg.SavePath = abs, ""
	cfg.Schema, cfg.Metadata, cfg.DocPresets = &models.ImportFile{}, hints, nil
	cfg.Validators = append(slices.Clone(MovelooperValidators), importsResolve(abs))
	cfg.PathFields = importFields(base)
	cfg.Trace = editor.Trace{}
	return cfg, nil
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
