package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/hints"
	"github.com/lucasassuncao/yedit/editor"
)

func buildMovelooperHints() (editor.MetadataSource, error) {
	return hints.Build()
}
