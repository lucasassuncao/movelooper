package cmd

import (
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/metadata"
	"github.com/lucasassuncao/yedit/spec"
)

func buildMovelooperHints() (spec.MetadataSource, error) {
	return metadata.New(models.Config{})
}
