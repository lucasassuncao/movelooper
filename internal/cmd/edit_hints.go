package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/metadata"
)

func buildMovelooperHints() (editor.MetadataSource, error) {
	return metadata.BuildFromProvider(models.Config{})
}
