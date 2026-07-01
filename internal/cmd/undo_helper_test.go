package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/history"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRestoreEntries_SkipsArchiveBatch(t *testing.T) {
	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	entries := []history.Entry{{
		Source:      "/src",
		Destination: "/dst/images.zip",
		Action:      string(models.ActionArchive),
		BatchID:     "batch_x",
		Category:    "images",
	}}
	restored := restoreEntries(context.Background(), m, entries)
	assert.Empty(t, restored, "archive entries are not restored")
	assert.Contains(t, buf.String(), "archive")
}
