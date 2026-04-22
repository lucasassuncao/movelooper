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

	data, err := os.ReadFile(absPath) //#nosec G304 -- absPath resolved via filepath.Abs
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", absPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", absPath, err)
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return data, nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%q: expected a YAML mapping at top level", absPath)
	}

	visited := map[string]bool{absPath: true}

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
	if importKeyIdx < 0 && len(importPaths) == 0 {
		return data, nil
	}

	// Strip the `import:` key-value pair.
	if importKeyIdx >= 0 {
		root.Content = append(root.Content[:importKeyIdx], root.Content[importKeyIdx+2:]...)
	}

	if len(importPaths) == 0 {
		return yaml.Marshal(&doc)
	}

	// Ensure a `categories:` sequence exists in the main document.
	if categoriesValNode == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "categories"}
		seqNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content, keyNode, seqNode)
		categoriesValNode = seqNode
	}

	baseDir := filepath.Dir(absPath)
	for _, imp := range importPaths {
		impAbs, err := filepath.Abs(filepath.Join(baseDir, imp))
		if err != nil {
			return nil, fmt.Errorf("resolving import %q declared in %q: %w", imp, absPath, err)
		}
		items, err := loadImportedCategories(impAbs, visited, []string{absPath})
		if err != nil {
			return nil, fmt.Errorf("importing %q: %w", imp, err)
		}
		categoriesValNode.Content = append(categoriesValNode.Content, items...)
	}

	return yaml.Marshal(&doc)
}

// loadImportedCategories reads a YAML file, resolves its own `import:` entries
// recursively, and returns the merged list of category sequence nodes.
func loadImportedCategories(absPath string, visited map[string]bool, chain []string) ([]*yaml.Node, error) {
	if visited[absPath] {
		return nil, fmt.Errorf("circular import detected: %s", strings.Join(append(chain, absPath), " → "))
	}
	visited[absPath] = true

	data, err := os.ReadFile(absPath) //#nosec G304 -- absPath resolved via filepath.Abs
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", absPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", absPath, err)
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return nil, nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%q: expected a YAML mapping at top level", absPath)
	}

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

	// Recurse into nested imports.
	baseDir := filepath.Dir(absPath)
	for _, imp := range importPaths {
		impAbs, err := filepath.Abs(filepath.Join(baseDir, imp))
		if err != nil {
			return nil, fmt.Errorf("resolving import %q declared in %q: %w", imp, absPath, err)
		}
		nested, err := loadImportedCategories(impAbs, visited, append(chain, absPath))
		if err != nil {
			return nil, fmt.Errorf("importing %q: %w", imp, err)
		}
		categoryItems = append(categoryItems, nested...)
	}

	return categoryItems, nil
}
