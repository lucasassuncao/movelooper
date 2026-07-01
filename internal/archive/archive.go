// Package archive packs a set of on-disk files into a single compressed
// archive (zip or tar.gz). It is agnostic of movelooper's config and tokens: it
// takes explicit (source, entry-name) pairs and writes the archive atomically.
package archive

import (
	"context"
	"fmt"
	"os"
)

// Format identifies the archive container/compression combination.
type Format string

const (
	FormatZip   Format = "zip"
	FormatTarGz Format = "tar.gz"
)

// Compression selects the effort/speed trade-off. Empty means CompressionBest.
type Compression string

const (
	CompressionNone Compression = "none"
	CompressionFast Compression = "fast"
	CompressionBest Compression = "best"
)

// Entry is one file to add: an absolute on-disk Source and the slash-separated
// Name it takes inside the archive.
type Entry struct {
	Source string
	Name   string
}

// Options configure a Write call.
type Options struct {
	Format      Format
	Compression Compression
	// OnProgress, when set, is called after each entry is written with the number
	// of entries done so far and the total. It runs synchronously on the writing
	// goroutine, so keep it fast. nil disables progress reporting.
	OnProgress func(done, total int)
}

// Extension returns the filename extension for f, including the leading dot.
func Extension(f Format) string {
	if f == FormatTarGz {
		return ".tar.gz"
	}
	return ".zip"
}

// Write creates an archive at destPath containing entries, atomically: it writes
// to destPath+".tmp", fsyncs, then renames into place. Each entry is streamed
// (never fully buffered). On any error no file is left at destPath. ctx cancels
// between entries.
func Write(ctx context.Context, destPath string, entries []Entry, opts Options) (retErr error) {
	tmp := destPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //#nosec G304 -- destPath derives from the configured destination directory
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if retErr != nil {
			if !closed {
				_ = f.Close()
			}
			_ = os.Remove(tmp)
		}
	}()

	switch opts.Format {
	case FormatZip:
		err = writeZip(ctx, f, entries, opts.Compression, opts.OnProgress)
	case FormatTarGz:
		err = writeTarGz(ctx, f, entries, opts.Compression, opts.OnProgress)
	default:
		err = fmt.Errorf("unknown archive format %q", opts.Format)
	}
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	closed = true
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, destPath)
}
