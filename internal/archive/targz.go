package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
)

func writeTarGz(ctx context.Context, w io.Writer, entries []Entry, comp Compression, onProgress func(done, total int)) error {
	gzw, err := gzip.NewWriterLevel(w, gzipLevel(comp))
	if err != nil {
		return err
	}
	tw := tar.NewWriter(gzw)
	for i, e := range entries {
		if err := ctx.Err(); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return err
		}
		if err := addTarEntry(tw, e); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return err
		}
		if onProgress != nil {
			onProgress(i+1, len(entries))
		}
	}
	if err := tw.Close(); err != nil {
		_ = gzw.Close()
		return err
	}
	return gzw.Close()
}

func addTarEntry(tw *tar.Writer, e Entry) error {
	src, err := os.Open(e.Source) //#nosec G304 -- source paths come from the directory scan
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = e.Name
	hdr.Format = tar.FormatPAX // PAX records carry UTF-8 names/metadata
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	return copyTarBody(tw, src, hdr.Size)
}

// copyTarBody copies exactly size bytes, the header's: a file still being
// written grows meanwhile, and the tar writer refuses anything past the size.
func copyTarBody(tw *tar.Writer, src io.Reader, size int64) error {
	_, err := io.CopyN(tw, src, size)
	return err
}

func gzipLevel(comp Compression) int {
	switch comp {
	case CompressionNone:
		return gzip.NoCompression
	case CompressionFast:
		return gzip.BestSpeed
	default: // best and empty
		return gzip.BestCompression
	}
}
