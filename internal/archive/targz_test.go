package archive

import (
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipLevel(t *testing.T) {
	tests := []struct {
		comp  Compression
		level int
	}{
		{CompressionNone, gzip.NoCompression},
		{CompressionFast, gzip.BestSpeed},
		{CompressionBest, gzip.BestCompression},
		{"", gzip.BestCompression},
	}
	for _, tt := range tests {
		t.Run(string(tt.comp), func(t *testing.T) {
			assert.Equal(t, tt.level, gzipLevel(tt.comp))
		})
	}
}

func TestWrite_TarGzCompressionNoneIsLarger(t *testing.T) {
	a := writeSource(t, t.TempDir(), "a.txt", strings.Repeat("a", 64*1024))
	size := func(comp Compression) int64 {
		dst := filepath.Join(t.TempDir(), "out.tar.gz")
		require.NoError(t, Write(context.Background(), dst, []Entry{{Source: a, Name: "a.txt"}},
			Options{Format: FormatTarGz, Compression: comp}))
		info, err := os.Stat(dst)
		require.NoError(t, err)
		return info.Size()
	}
	assert.Greater(t, size(CompressionNone), size(CompressionBest))
}

func TestWrite_TarGzKeepsUTF8AndNestedNames(t *testing.T) {
	a := writeSource(t, t.TempDir(), "a.txt", "AAA")
	dst := filepath.Join(t.TempDir(), "out.tar.gz")
	name := "sub/relatório, ação 2026.txt"

	require.NoError(t, Write(context.Background(), dst, []Entry{{Source: a, Name: name}},
		Options{Format: FormatTarGz}))

	assert.Equal(t, "AAA", readTarGz(t, dst)[name])
}

func TestWrite_TarGzMissingSourceLeavesNoArchive(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.tar.gz")
	err := Write(context.Background(), dst, []Entry{{Source: "/does/not/exist", Name: "x"}},
		Options{Format: FormatTarGz})
	require.Error(t, err)
	assert.NoFileExists(t, dst)
	assert.NoFileExists(t, dst+".tmp")
}

func TestWrite_TarGzCanceledContextLeavesNoArchive(t *testing.T) {
	a := writeSource(t, t.TempDir(), "a.txt", "A")
	dst := filepath.Join(t.TempDir(), "out.tar.gz")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Write(ctx, dst, []Entry{{Source: a, Name: "a.txt"}}, Options{Format: FormatTarGz})
	require.ErrorIs(t, err, context.Canceled)
	assert.NoFileExists(t, dst)
	assert.NoFileExists(t, dst+".tmp")
}
