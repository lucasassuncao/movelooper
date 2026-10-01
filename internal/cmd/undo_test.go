package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/logger"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUndoCmd_NilHistoryFails(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	m := &models.Movelooper{Logger: logger.NewSlog(&buf, "info", false)}
	cmd := UndoCmd(m)
	cmd.SetArgs([]string{})

	err := cmd.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "history tracking is not initialized")
}

// With no batches there is nothing to pick, so the picker must never open.
func TestUndoCmd_EmptyHistoryIsANoop(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	m := newBufMovelooper(t, &buf, nil)
	cmd := UndoCmd(m)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.ExecuteContext(context.Background()))
	assert.Contains(t, buf.String(), "no batches in history")
}
