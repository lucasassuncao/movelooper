package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtension(t *testing.T) {
	assert.Equal(t, ".zip", Extension(FormatZip))
	assert.Equal(t, ".tar.gz", Extension(FormatTarGz))
}

func TestWrite_UnknownFormat(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.bin")
	err := Write(context.Background(), dst, nil, Options{Format: "rar"})
	require.Error(t, err)
	assert.NoFileExists(t, dst)
}

func writeSource(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func readZip(t *testing.T, path string) map[string]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer r.Close()
	out := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		out[f.Name] = string(b)
	}
	return out
}

func readTarGz(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		b, err := io.ReadAll(tr)
		require.NoError(t, err)
		out[h.Name] = string(b)
	}
	return out
}

func TestWrite_ZipRoundTrip(t *testing.T) {
	src := t.TempDir()
	// a non-ASCII name with spaces/commas exercises the UTF-8 flag
	a := writeSource(t, src, "a.txt", "AAA")
	b := writeSource(t, src, "ChatGPT Image, 2026.png", "BBB")
	dst := filepath.Join(t.TempDir(), "out.zip")

	err := Write(context.Background(), dst, []Entry{
		{Source: a, Name: "a.txt"},
		{Source: b, Name: "sub/ChatGPT Image, 2026.png"},
	}, Options{Format: FormatZip, Compression: CompressionBest})
	require.NoError(t, err)

	got := readZip(t, dst)
	assert.Equal(t, "AAA", got["a.txt"])
	assert.Equal(t, "BBB", got["sub/ChatGPT Image, 2026.png"], "slash-separated nested entry preserved")
	assert.NoFileExists(t, dst+".tmp", "temp file removed after rename")
}

func TestWrite_TarGzRoundTrip(t *testing.T) {
	src := t.TempDir()
	a := writeSource(t, src, "a.txt", "AAA")
	dst := filepath.Join(t.TempDir(), "out.tar.gz")

	err := Write(context.Background(), dst, []Entry{{Source: a, Name: "a.txt"}},
		Options{Format: FormatTarGz, Compression: CompressionFast})
	require.NoError(t, err)

	got := readTarGz(t, dst)
	assert.Equal(t, "AAA", got["a.txt"])
}

func TestWrite_OnProgress(t *testing.T) {
	src := t.TempDir()
	a := writeSource(t, src, "a.txt", "A")
	b := writeSource(t, src, "b.txt", "B")
	dst := filepath.Join(t.TempDir(), "out.zip")

	var calls [][2]int
	err := Write(context.Background(), dst, []Entry{{Source: a, Name: "a.txt"}, {Source: b, Name: "b.txt"}},
		Options{Format: FormatZip, OnProgress: func(done, total int) {
			calls = append(calls, [2]int{done, total})
		}})
	require.NoError(t, err)
	require.Len(t, calls, 2, "called once per entry")
	assert.Equal(t, [2]int{1, 2}, calls[0])
	assert.Equal(t, [2]int{2, 2}, calls[1], "last call reports completion")
}

func TestWrite_MissingSourceLeavesNoArchive(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.zip")
	err := Write(context.Background(), dst, []Entry{{Source: "/does/not/exist", Name: "x"}},
		Options{Format: FormatZip})
	require.Error(t, err)
	assert.NoFileExists(t, dst)
	assert.NoFileExists(t, dst+".tmp")
}
