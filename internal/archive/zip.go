package archive

import (
	"archive/zip"
	"compress/flate"
	"context"
	"io"
	"os"
)

func writeZip(ctx context.Context, w io.Writer, entries []Entry, comp Compression, onProgress func(done, total int)) error {
	zw := zip.NewWriter(w)
	if lvl, deflate := deflateLevel(comp); deflate {
		zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, lvl)
		})
	}
	for i, e := range entries {
		if err := ctx.Err(); err != nil {
			_ = zw.Close()
			return err
		}
		if err := addZipEntry(zw, e, comp); err != nil {
			_ = zw.Close()
			return err
		}
		if onProgress != nil {
			onProgress(i+1, len(entries))
		}
	}
	return zw.Close()
}

func addZipEntry(zw *zip.Writer, e Entry, comp Compression) error {
	info, err := os.Stat(e.Source)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = e.Name // already slash-separated by the caller
	hdr.Modified = info.ModTime()
	if comp == CompressionNone {
		hdr.Method = zip.Store
	} else {
		hdr.Method = zip.Deflate
	}
	// archive/zip sets the UTF-8 name flag automatically when Name is not plain
	// ASCII, because hdr.NonUTF8 is left false.
	fw, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	src, err := os.Open(e.Source) //#nosec G304 -- source paths come from the directory scan
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(fw, src)
	return err
}

// deflateLevel maps Compression to a flate level. The bool is false for
// CompressionNone (store, no deflate registration needed).
func deflateLevel(comp Compression) (int, bool) {
	switch comp {
	case CompressionNone:
		return 0, false
	case CompressionFast:
		return flate.BestSpeed, true
	default: // best and empty
		return flate.BestCompression, true
	}
}
