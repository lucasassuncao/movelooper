package cmd

import (
	"testing"

	"github.com/lucasassuncao/yedit/document"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMovelooperValidatorsAgainstSampleConfig guards the repository's sample
// config against the cross-field rules. The FromMetadata validators are inert
// outside editor.Run (they are wired by the editor session); their engine is
// covered by yedit's own tests and the hint markers by edit_hints_test.go.
func TestMovelooperValidatorsAgainstSampleConfig(t *testing.T) {
	t.Parallel()
	doc, err := document.Load("../../movelooper.yaml", nil)
	require.NoError(t, err)
	assert.Empty(t, editor.RunAll(MovelooperValidators, doc.Raw(), doc.Blocks()))
}

// TestMovelooperValidatorsCatchBrokenConfig exercises the explicit cross-field
// rules with a config that violates them.
func TestMovelooperValidatorsCatchBrokenConfig(t *testing.T) {
	t.Parallel()
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
	// duplicate name, regex+glob in match, age.min >= age.max,
	// any+all+match at same filter level → 3 pair violations (any/all, any/match, all/match)
	assert.GreaterOrEqual(t, len(errs), 6)
}
