package fileops

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ConflictArgs carries the paths needed by a ConflictResolver.
type ConflictArgs struct {
	Src      string
	Dst      string
	DestDir  string
	FileName string
}

// ConflictResolver resolves a naming conflict when a destination file already exists.
// Resolve returns the final destination path, whether the move should proceed, and
// any error encountered. When shouldMove is false the caller must skip the file.
// SkipMessage returns the log message to emit when shouldMove is false; "" means no log.
type ConflictResolver interface {
	Resolve(args ConflictArgs) (resolvedPath string, shouldMove bool, err error)
	SkipMessage() string
}

// conflictResolvers maps strategy names to their ConflictResolver implementation.
var conflictResolvers = map[string]ConflictResolver{
	"rename":     &renameResolver{},
	"overwrite":  &overwriteResolver{},
	"skip":       &skipResolver{},
	"hash_check": &hashCheckResolver{},
	"newest":     &newestResolver{},
	"oldest":     &oldestResolver{},
	"larger":     &largerResolver{},
	"smaller":    &smallerResolver{},
}

type renameResolver struct{}

func (r *renameResolver) Resolve(args ConflictArgs) (string, bool, error) {
	path, err := getUniqueDestinationPath(args.DestDir, args.FileName)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

func (r *renameResolver) SkipMessage() string { return "" }

type overwriteResolver struct{}

func (r *overwriteResolver) Resolve(args ConflictArgs) (string, bool, error) {
	if err := os.Remove(args.Dst); err != nil {
		return "", false, fmt.Errorf("failed to remove destination file for overwrite: %w", err)
	}
	return args.Dst, true, nil
}

func (r *overwriteResolver) SkipMessage() string { return "" }

type skipResolver struct{}

func (r *skipResolver) Resolve(_ ConflictArgs) (string, bool, error) {
	return "", false, nil
}

func (r *skipResolver) SkipMessage() string { return "file skipped due to conflict strategy" }

func compareFileHashes(file1, file2 string) (bool, error) {
	h1, err := calculateHash(file1)
	if err != nil {
		return false, err
	}
	h2, err := calculateHash(file2)
	if err != nil {
		return false, err
	}
	return h1 == h2, nil
}

func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filepath.Clean(filePath)) //#nosec G304 -- path comes from directory walk
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

type newestResolver struct{}

func (r *newestResolver) Resolve(args ConflictArgs) (string, bool, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, err
	}
	if !srcInfo.ModTime().After(dstInfo.ModTime()) {
		return "", false, nil
	}
	if err := os.Remove(args.Dst); err != nil {
		return "", false, fmt.Errorf("newest: failed to remove older destination: %w", err)
	}
	return args.Dst, true, nil
}

func (r *newestResolver) SkipMessage() string { return "file skipped - destination is newer" }

type oldestResolver struct{}

func (r *oldestResolver) Resolve(args ConflictArgs) (string, bool, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, err
	}
	if !srcInfo.ModTime().Before(dstInfo.ModTime()) {
		return "", false, nil
	}
	if err := os.Remove(args.Dst); err != nil {
		return "", false, fmt.Errorf("oldest: failed to remove newer destination: %w", err)
	}
	return args.Dst, true, nil
}

func (r *oldestResolver) SkipMessage() string { return "file skipped - destination is older" }

type largerResolver struct{}

func (r *largerResolver) Resolve(args ConflictArgs) (string, bool, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, err
	}
	if srcInfo.Size() <= dstInfo.Size() {
		return "", false, nil
	}
	if err := os.Remove(args.Dst); err != nil {
		return "", false, fmt.Errorf("larger: failed to remove smaller destination: %w", err)
	}
	return args.Dst, true, nil
}

func (r *largerResolver) SkipMessage() string { return "file skipped - destination is larger" }

type smallerResolver struct{}

func (r *smallerResolver) Resolve(args ConflictArgs) (string, bool, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, err
	}
	if srcInfo.Size() >= dstInfo.Size() {
		return "", false, nil
	}
	if err := os.Remove(args.Dst); err != nil {
		return "", false, fmt.Errorf("smaller: failed to remove larger destination: %w", err)
	}
	return args.Dst, true, nil
}

func (r *smallerResolver) SkipMessage() string { return "file skipped - destination is smaller" }

type hashCheckResolver struct{}

func (r *hashCheckResolver) Resolve(args ConflictArgs) (string, bool, error) {
	match, err := compareFileHashes(args.Src, args.Dst)
	if err != nil {
		return "", false, err
	}
	if match {
		if err := os.Remove(args.Src); err != nil {
			return "", false, fmt.Errorf("failed to remove duplicate source file: %w", err)
		}
		return "", false, nil
	}
	path, err := getUniqueDestinationPath(args.DestDir, args.FileName)
	if err != nil {
		return "", false, err
	}
	return path, true, nil
}

func (r *hashCheckResolver) SkipMessage() string { return "duplicate file removed from source" }
