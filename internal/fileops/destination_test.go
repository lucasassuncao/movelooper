package fileops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDestTokenContext(t *testing.T) *tokens.TokenContext {
	t.Helper()
	src := filepath.Join(t.TempDir(), "photo.jpg")
	writeFile(t, src, []byte("x"))
	info, err := os.Stat(src)
	require.NoError(t, err)
	return &tokens.TokenContext{Info: info, CategoryName: "photos", Now: time.Now(), SourcePath: src}
}

func TestResolveDestDir(t *testing.T) {
	t.Parallel()
	dst := filepath.Join(t.TempDir(), "dst")

	tests := []struct {
		name       string
		organizeBy string
		want       string
	}{
		{"no organize-by is the destination itself", "", dst},
		{"organize-by adds a subdirectory", "{category}", filepath.Join(dst, "photos")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat := &models.Category{Name: "photos", Destination: models.CategoryDestination{Path: dst, OrganizeBy: tt.organizeBy}}
			assert.Equal(t, tt.want, ResolveDestDir(cat, newDestTokenContext(t)))
		})
	}
}

// The rename's seq tokens scan tctx.DestDir, so it must be set before renaming.
func TestResolveDestination_SetsDestDirBeforeRename(t *testing.T) {
	t.Parallel()
	dst := filepath.Join(t.TempDir(), "dst")
	cat := &models.Category{Name: "photos", Destination: models.CategoryDestination{
		Path: dst, OrganizeBy: "{category}", Rename: "{category}_copy",
	}}
	tctx := newDestTokenContext(t)

	dir, name := ResolveDestination(cat, tctx)
	assert.Equal(t, filepath.Join(dst, "photos"), dir)
	assert.Equal(t, dir, tctx.DestDir)
	assert.Equal(t, "photos_copy", name)
	assert.NoDirExists(t, dst, "resolving must not create the destination")
}

func TestResolveDestination_EmptyRenameKeepsTheName(t *testing.T) {
	t.Parallel()
	cat := &models.Category{Destination: models.CategoryDestination{Path: t.TempDir()}}

	_, name := ResolveDestination(cat, newDestTokenContext(t))
	assert.Equal(t, "photo.jpg", name)
}
