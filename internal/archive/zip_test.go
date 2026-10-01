package archive

import (
	"archive/zip"
	"compress/flate"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeflateLevel(t *testing.T) {
	tests := []struct {
		comp    Compression
		level   int
		deflate bool
	}{
		{CompressionNone, 0, false},
		{CompressionFast, flate.BestSpeed, true},
		{CompressionBest, flate.BestCompression, true},
		{"", flate.BestCompression, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.comp), func(t *testing.T) {
			level, deflate := deflateLevel(tt.comp)
			assert.Equal(t, tt.level, level)
			assert.Equal(t, tt.deflate, deflate)
		})
	}
}

func TestWrite_ZipMethodFollowsCompression(t *testing.T) {
	tests := []struct {
		comp   Compression
		method uint16
	}{
		{CompressionNone, zip.Store},
		{CompressionFast, zip.Deflate},
		{CompressionBest, zip.Deflate},
	}
	for _, tt := range tests {
		t.Run(string(tt.comp), func(t *testing.T) {
			a := writeSource(t, t.TempDir(), "a.txt", "AAA")
			dst := filepath.Join(t.TempDir(), "out.zip")
			require.NoError(t, Write(context.Background(), dst, []Entry{{Source: a, Name: "a.txt"}},
				Options{Format: FormatZip, Compression: tt.comp}))

			r, err := zip.OpenReader(dst)
			require.NoError(t, err)
			defer r.Close()
			require.Len(t, r.File, 1)
			assert.Equal(t, tt.method, r.File[0].Method)
			assert.Equal(t, "AAA", readZip(t, dst)["a.txt"])
		})
	}
}

func TestWrite_ZipKeepsModTime(t *testing.T) {
	a := writeSource(t, t.TempDir(), "a.txt", "A")
	mtime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(a, mtime, mtime))
	dst := filepath.Join(t.TempDir(), "out.zip")

	require.NoError(t, Write(context.Background(), dst, []Entry{{Source: a, Name: "a.txt"}},
		Options{Format: FormatZip}))

	r, err := zip.OpenReader(dst)
	require.NoError(t, err)
	defer r.Close()
	// the DOS timestamp field only has 2-second precision
	assert.WithinDuration(t, mtime, r.File[0].Modified, 2*time.Second)
}

func TestWrite_ZipCanceledContextLeavesNoArchive(t *testing.T) {
	a := writeSource(t, t.TempDir(), "a.txt", "A")
	dst := filepath.Join(t.TempDir(), "out.zip")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Write(ctx, dst, []Entry{{Source: a, Name: "a.txt"}}, Options{Format: FormatZip})
	require.ErrorIs(t, err, context.Canceled)
	assert.NoFileExists(t, dst)
	assert.NoFileExists(t, dst+".tmp")
}
