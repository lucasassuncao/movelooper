package tokens

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGroupBy_Mime(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "photo.jpg")
	require.NoError(t, os.WriteFile(png, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644))
	info, err := os.Stat(png)
	require.NoError(t, err)

	ctx := &TokenContext{Info: info, Now: time.Now(), SourcePath: png}
	assert.Equal(t, filepath.FromSlash("image/png"), ResolveGroupBy("{mime-type}/{mime-ext}", ctx))
}

func TestValidateTemplate_Mime(t *testing.T) {
	assert.NoError(t, ValidateTemplate("{mime-type}/{mime-ext}"))
	assert.NoError(t, ValidateTemplate("{mime}"))
}
