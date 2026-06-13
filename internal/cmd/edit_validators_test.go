package cmd

import (
	"testing"

	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/editor"
)

// TestMovelooperValidatorsAgainstSampleConfig guards the repository's sample
// config against the cross-field rules. The FromMetadata validators are inert
// outside editor.Run (they are wired by the editor session); their engine is
// covered by yedit's own tests and the hint markers by edit_hints_test.go.
func TestMovelooperValidatorsAgainstSampleConfig(t *testing.T) {
	doc, err := document.Load("../../movelooper.yaml", nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if errs := editor.RunAll(MovelooperValidators, doc.Raw(), doc.Blocks()); len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("violation: %s", e.String())
		}
	}
}

// TestMovelooperValidatorsCatchBrokenConfig exercises the explicit cross-field
// rules with a config that violates them.
func TestMovelooperValidatorsCatchBrokenConfig(t *testing.T) {
	raw := []byte(`
categories:
  - name: dup
    source:
      path: a
      extensions: [pdf]
      filter:
        any:
          - match:
              regex: "x"
              glob: "y"
        all:
          - age:
              min: 48h
              max: 24h
        match:
          glob: "*.pdf"
    destination:
      path: b
  - name: dup
    source:
      path: c
      extensions: [pdf]
    destination:
      path: d
`)
	errs := editor.RunAll(MovelooperValidators, raw, nil)
	for _, e := range errs {
		t.Logf("violation: %s", e.String())
	}
	// duplicate name, regex+glob in match, age.min >= age.max,
	// any+all+match at same filter level → 3 pair violations (any/all, any/match, all/match)
	wantAtLeast := 6
	if len(errs) < wantAtLeast {
		t.Errorf("expected at least %d violations, got %d", wantAtLeast, len(errs))
	}
}
