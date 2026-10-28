package hooks

import (
	"bytes"
	"context"
	"github.com/lucasassuncao/movelooper/internal/logger"
	"io"
	"strings"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cmd reads its own quotes: Go escaped the inner ones with backslashes, which
// cmd printed as they were and passed on to the programs it ran.
func TestRunHook_CmdKeepsQuotesAsWritten(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	hook := &models.CategoryHook{Shell: "cmd", OnFailure: "abort", Run: []string{`echo "a b"`}}
	require.NoError(t, RunHook(context.Background(), hook, HookContext{Log: logger.Discard(), Stdout: &out, Stderr: io.Discard}, nil))
	assert.Equal(t, `"a b"`, strings.TrimSpace(out.String()))
}
