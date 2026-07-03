package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, data, 0o644))
	return p
}

func TestDetect(t *testing.T) {
	png := write(t, "x.bin", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	got, err := Detect(png)
	require.NoError(t, err)
	assert.Equal(t, "image/png", got.Full)
	assert.Equal(t, "image", got.Type)
	assert.Equal(t, "png", got.Ext)

	pdf := write(t, "x.txt", []byte("%PDF-1.5\n%rest"))
	got, err = Detect(pdf)
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", got.Full)
	assert.Equal(t, "application", got.Type)
	assert.Equal(t, "pdf", got.Ext)

	txt := write(t, "note.dat", []byte("just some ascii text\n"))
	got, err = Detect(txt)
	require.NoError(t, err)
	assert.Equal(t, "text", got.Type, "plain text detected regardless of extension")
}

func TestDetect_MissingFile(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "nope"))
	require.Error(t, err)
}
