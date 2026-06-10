// Package hints provides HintNode and BuildFrom for building an editor.HintSource
// from a hierarchical tree of field metadata. Use BuildFrom to reflect over
// a schema struct and derive the Type field for each node automatically.
package hints

import (
	"reflect"
	"strings"
	"time"

	"github.com/lucasassuncao/yedit/editor"
)

// HintNode represents a single field's hint data plus its children.
// Embed FieldMeta for Description, Type, Required, Default, OneOf, Example.
// Use shared pointers in Children to model recursive schema types without
// duplicating definitions (e.g. CategoryFilter.Any []CategoryFilter).
type HintNode struct {
	editor.FieldMeta
	Children map[string]*HintNode
}

// BuildFrom reflects over schema (a pointer to the root config struct) to
// derive Type for each node in tree, then returns an editor.HintSource that
// resolves FieldHint calls by walking the HintNode graph.
//
// Fields tagged yaml:"-" are skipped. The schema argument is used only for
// type reflection; it is not read for Required/OneOf/Default.
func BuildFrom(schema any, tree map[string]*HintNode) editor.HintSource {
	rootType := reflect.TypeOf(schema)
	visited := map[*HintNode]bool{}
	for blockName, node := range tree {
		ft := fieldTypeByYAML(rootType, blockName)
		fillTypes(node, ft, visited)
	}
	return &hintSource{tree: tree}
}

// hintSource implements editor.HintSource backed by a HintNode tree.
type hintSource struct {
	tree map[string]*HintNode
}

func (h *hintSource) FieldHint(block, fieldPath string) editor.FieldMeta {
	node, ok := h.tree[block]
	if !ok {
		return editor.FieldMeta{}
	}
	if fieldPath == "" {
		return node.FieldMeta
	}
	segments := strings.Split(fieldPath, ".")
	cur := node
	for _, seg := range segments {
		if cur.Children == nil {
			return editor.FieldMeta{}
		}
		next, ok := cur.Children[seg]
		if !ok {
			return editor.FieldMeta{}
		}
		cur = next
	}
	return cur.FieldMeta
}

// fillTypes performs a DFS over the HintNode graph, setting Type on each node
// by reflecting over the corresponding Go type. visited prevents infinite loops
// on cyclic graphs (shared-pointer children like filterChildren).
func fillTypes(node *HintNode, t reflect.Type, visited map[*HintNode]bool) {
	if node == nil || visited[node] {
		return
	}
	visited[node] = true

	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t != nil {
		node.Type = goTypeString(t)
	}

	// Resolve element type for slices (e.g. []Category → Category).
	elem := t
	if elem != nil && elem.Kind() == reflect.Slice {
		elem = elem.Elem()
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
	}

	for childName, childNode := range node.Children {
		var childType reflect.Type
		if elem != nil {
			childType = fieldTypeByYAML(elem, childName)
		}
		fillTypes(childNode, childType, visited)
	}
}

// fieldTypeByYAML finds the Go type of the field with yaml tag name yamlName
// in struct type t. Returns nil if not found or t is not a struct.
func fieldTypeByYAML(t reflect.Type, yamlName string) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "-" {
			continue
		}
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		if name == yamlName {
			return f.Type
		}
	}
	return nil
}

var durationType = reflect.TypeOf(time.Duration(0))

// goTypeString converts a reflect.Type to a human-readable type label shown
// in the hint panel.
func goTypeString(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t == durationType {
			return "duration"
		}
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Slice:
		elem := t.Elem()
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			return "[]object"
		}
		return "[]" + goTypeString(elem)
	case reflect.Map:
		return "map[" + goTypeString(t.Key()) + "]" + goTypeString(t.Elem())
	case reflect.Struct:
		return "object"
	case reflect.Interface:
		return "any"
	default:
		return t.Kind().String()
	}
}
