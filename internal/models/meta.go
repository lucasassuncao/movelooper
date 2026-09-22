package models

// meta is a field's metadata as movelooper declares it. It exists so the
// entries of a Metadata() map stay compiler-checked: a mistyped field does not
// build, where a mistyped map key would only fail at startup.
//
// It carries no import. Whoever reads Metadata() - the editor, the doc
// generator - marshals the map and decodes it by these yaml names, so the
// names here are the contract, not the type. They mirror yedit's
// spec.FieldMeta; a name with no counterpart there fails at startup.
type meta struct {
	Description string   `yaml:"description,omitempty"`
	Type        string   `yaml:"type,omitempty"` // derived from the Go type when empty
	Required    bool     `yaml:"required,omitempty"`
	Default     string   `yaml:"default,omitempty"`
	Example     string   `yaml:"example,omitempty"`
	OneOf       []string `yaml:"oneof,omitempty"`
	NotOneOf    []string `yaml:"notoneof,omitempty"`

	// Value constraints, enforced by the FromMetadata validators.
	Min       string   `yaml:"min,omitempty"`
	Max       string   `yaml:"max,omitempty"`
	Pattern   string   `yaml:"pattern,omitempty"` // RE2
	MinLength int      `yaml:"minlength,omitempty"`
	MaxLength int      `yaml:"maxlength,omitempty"`
	Formats   []string `yaml:"formats,omitempty"` // format names, OR semantics

	// Collection constraints.
	MinCount int  `yaml:"mincount,omitempty"`
	MaxCount int  `yaml:"maxcount,omitempty"`
	Unique   bool `yaml:"unique,omitempty"`

	// Non-empty marks the field deprecated; the value is the migration hint.
	Deprecated string `yaml:"deprecated,omitempty"`

	// Editor-only. docgen ignores these.
	Presentation string `yaml:"presentation,omitempty"` // "flat", "inline", "overlay"
	Multiline    bool   `yaml:"multiline,omitempty"`
	Snippet      string `yaml:"snippet,omitempty"`
	PreChecked   bool   `yaml:"prechecked,omitempty"`
}
