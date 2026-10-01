package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveImports reads the YAML file at path, recursively resolves any top-level
// `import:` entries, merges all `categories:` items into the main document, and
// returns the final merged YAML bytes ready to be fed into Viper.
// The `import:` key is stripped from the output.
// Import paths are relative to the file that declares them.
// Circular imports are detected and reported as errors.
func ResolveImports(path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", path, err)
	}

	data, doc, err := parseYAMLFile(absPath)
	if err != nil || doc == nil {
		return data, err
	}
	root := doc.Content[0]

	var importPaths []string
	var categoriesValNode *yaml.Node
	importKeyIdx := -1

	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "import":
			if err := root.Content[i+1].Decode(&importPaths); err != nil {
				return nil, fmt.Errorf("%q: decoding import list: %w", absPath, err)
			}
			importKeyIdx = i
		case "categories":
			categoriesValNode = root.Content[i+1]
		}
	}

	// Nothing to do.
	if importKeyIdx < 0 {
		return data, nil
	}

	// Strip the `import:` key-value pair.
	root.Content = append(root.Content[:importKeyIdx], root.Content[importKeyIdx+2:]...)

	if len(importPaths) == 0 {
		return yaml.Marshal(doc)
	}

	// Ensure a `categories:` sequence exists in the main document.
	if categoriesValNode == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "categories"}
		seqNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content, keyNode, seqNode)
		categoriesValNode = seqNode
	}

	merged := map[string]bool{absPath: true}
	items, err := importAll(absPath, importPaths, merged, []string{absPath})
	if err != nil {
		return nil, err
	}
	categoriesValNode.Content = append(categoriesValNode.Content, items...)

	return yaml.Marshal(doc)
}

// parseYAMLFile reads and parses the YAML file at absPath. doc is nil when the
// file is empty; otherwise its top level is guaranteed to be a mapping.
func parseYAMLFile(absPath string) ([]byte, *yaml.Node, error) {
	data, err := os.ReadFile(absPath) //#nosec G304 -- absPath resolved via filepath.Abs
	if err != nil {
		return nil, nil, fmt.Errorf("reading %q: %w", absPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("parsing %q: %w", absPath, err)
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return data, nil, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("%q: expected a YAML mapping at top level", absPath)
	}
	return data, &doc, nil
}

// importAll loads the categories of every path in importPaths, which are
// relative to declaringFile. chain is the import path ending at declaringFile.
func importAll(declaringFile string, importPaths []string, merged map[string]bool, chain []string) ([]*yaml.Node, error) {
	baseDir := filepath.Dir(declaringFile)
	var items []*yaml.Node
	for _, imp := range importPaths {
		impAbs, err := filepath.Abs(filepath.Join(baseDir, imp))
		if err != nil {
			return nil, fmt.Errorf("resolving import %q declared in %q: %w", imp, declaringFile, err)
		}
		nested, err := loadImportedCategories(impAbs, merged, chain)
		if err != nil {
			return nil, fmt.Errorf("importing %q: %w", imp, err)
		}
		items = append(items, nested...)
	}
	return items, nil
}

// loadImportedCategories reads a YAML file, resolves its own `import:` entries
// recursively, and returns the merged list of category sequence nodes.
func loadImportedCategories(absPath string, merged map[string]bool, chain []string) ([]*yaml.Node, error) {
	// A cycle exists only when the file is already an ancestor on the current
	// import path. Checking against the active chain (not a global visited set)
	// lets a file be shared by sibling imports (a diamond) without a false cycle.
	for _, ancestor := range chain {
		if ancestor == absPath {
			return nil, fmt.Errorf("circular import detected: %s", strings.Join(append(chain, absPath), " → "))
		}
	}
	// Already pulled in via another import path: skip to avoid duplicating its
	// categories. This is what distinguishes a diamond from a cycle.
	if merged[absPath] {
		return nil, nil
	}
	merged[absPath] = true

	_, doc, err := parseYAMLFile(absPath)
	if err != nil || doc == nil {
		return nil, err
	}
	root := doc.Content[0]

	var importPaths []string
	var categoryItems []*yaml.Node

	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "import":
			if err := root.Content[i+1].Decode(&importPaths); err != nil {
				return nil, fmt.Errorf("%q: decoding import list: %w", absPath, err)
			}
		case "categories":
			categoryItems = root.Content[i+1].Content
		}
	}

	nested, err := importAll(absPath, importPaths, merged, append(chain, absPath))
	if err != nil {
		return nil, err
	}
	return append(categoryItems, nested...), nil
}
