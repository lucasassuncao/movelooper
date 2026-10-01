package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocs_WritesEveryOutput(t *testing.T) {
	// Not parallel: generateDocs writes relative to the working directory.
	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	require.NoError(t, generateDocs(&buf))

	for _, dir := range []string{"attributes", "examples", "schema"} {
		entries, err := os.ReadDir(filepath.Join("docs", "movelooper", dir))
		require.NoError(t, err, dir)
		assert.NotEmpty(t, entries, "%s is empty", dir)
	}
	assert.Contains(t, buf.String(), "Documentation generated")
}
