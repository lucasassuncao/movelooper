package fileops

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lucasassuncao/movelooper/internal/models"
)

// ConflictArgs carries the paths needed by a ConflictResolver.
type ConflictArgs struct {
	Src      string
	Dst      string
	DestDir  string
	FileName string
}

// FinalizeFunc commits or rolls back a destination that a resolver moved aside
// before the file action ran. It is invoked once the action completes: when
// failed is true the original destination is restored, otherwise the set-aside
// copy is discarded. Resolvers that do not displace the destination return nil.
type FinalizeFunc func(failed bool) error

// ConflictResolver resolves a naming conflict when a destination file already exists.
// Resolve returns the final destination path, whether the move should proceed, an
// optional finalize callback (nil when the destination is left untouched), and any
// error encountered. When shouldMove is false the caller must skip the file.
// SkipMessage returns the log message to emit when shouldMove is false; "" means no log.
type ConflictResolver interface {
	Resolve(args ConflictArgs) (resolvedPath string, shouldMove bool, finalize FinalizeFunc, err error)
	SkipMessage() string
}

// swapAside renames an existing destination to a unique temporary backup and
// returns a FinalizeFunc that restores it when the action fails or removes it
// when the action succeeds. This lets a replace-style strategy recover the
// original file if the subsequent action fails partway through.
func swapAside(dst string) (FinalizeFunc, error) {
	backup, err := uniqueBackupPath(dst)
	if err != nil {
		return nil, err
	}
	if err := os.Rename(dst, backup); err != nil {
		return nil, err
	}
	return func(failed bool) error {
		if failed {
			_ = os.Remove(dst) // drop any partial output the failed action left behind
			return os.Rename(backup, dst)
		}
		return os.Remove(backup)
	}, nil
}

// uniqueBackupPath returns a path next to dst that does not yet exist.
func uniqueBackupPath(dst string) (string, error) {
	for i := 0; i < 10000; i++ {
		candidate := fmt.Sprintf("%s.ml-bak.%d", dst, i)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find a free backup name for %q", dst)
}

// conflictResolvers maps strategy names to their ConflictResolver implementation.
var conflictResolvers = map[models.ConflictStrategy]ConflictResolver{
	models.ConflictStrategyRename:    &renameResolver{},
	models.ConflictStrategyOverwrite: &overwriteResolver{},
	models.ConflictStrategySkip:      &skipResolver{},
	models.ConflictStrategyHashCheck: &hashCheckResolver{},
	models.ConflictStrategyNewest:    &newestResolver{},
	models.ConflictStrategyOldest:    &oldestResolver{},
	models.ConflictStrategyLarger:    &largerResolver{},
	models.ConflictStrategySmaller:   &smallerResolver{},
}

type renameResolver struct{}

func (r *renameResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	path, err := getUniqueDestinationPath(args.DestDir, args.FileName)
	if err != nil {
		return "", false, nil, err
	}
	return path, true, nil, nil
}

func (r *renameResolver) SkipMessage() string { return "" }

type overwriteResolver struct{}

func (r *overwriteResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	if runtime.GOOS == "windows" {
		// os.Rename fails on Windows when the destination exists. Move it aside
		// instead of deleting it, so a failed action can be rolled back.
		finalize, err := swapAside(args.Dst)
		if err != nil {
			return "", false, nil, fmt.Errorf("failed to set aside destination file for overwrite: %w", err)
		}
		return args.Dst, true, finalize, nil
	}
	// On POSIX, os.Rename(src, dst) atomically replaces an existing dst,
	// so no pre-removal is needed. Cross-device copyFile uses O_TRUNC which
	// also overwrites safely without a prior remove.
	return args.Dst, true, nil, nil
}

func (r *overwriteResolver) SkipMessage() string { return "" }

type skipResolver struct{}

func (r *skipResolver) Resolve(_ ConflictArgs) (string, bool, FinalizeFunc, error) {
	return "", false, nil, nil
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

func (r *newestResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, nil, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, nil, err
	}
	if !srcInfo.ModTime().After(dstInfo.ModTime()) {
		return "", false, nil, nil
	}
	finalize, err := swapAside(args.Dst)
	if err != nil {
		return "", false, nil, fmt.Errorf("newest: failed to set aside older destination: %w", err)
	}
	return args.Dst, true, finalize, nil
}

func (r *newestResolver) SkipMessage() string { return "file skipped - destination is newer" }

type oldestResolver struct{}

func (r *oldestResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, nil, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, nil, err
	}
	if !srcInfo.ModTime().Before(dstInfo.ModTime()) {
		return "", false, nil, nil
	}
	finalize, err := swapAside(args.Dst)
	if err != nil {
		return "", false, nil, fmt.Errorf("oldest: failed to set aside newer destination: %w", err)
	}
	return args.Dst, true, finalize, nil
}

func (r *oldestResolver) SkipMessage() string { return "file skipped - destination is older" }

type largerResolver struct{}

func (r *largerResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, nil, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, nil, err
	}
	if srcInfo.Size() <= dstInfo.Size() {
		return "", false, nil, nil
	}
	finalize, err := swapAside(args.Dst)
	if err != nil {
		return "", false, nil, fmt.Errorf("larger: failed to set aside smaller destination: %w", err)
	}
	return args.Dst, true, finalize, nil
}

func (r *largerResolver) SkipMessage() string { return "file skipped - destination is larger" }

type smallerResolver struct{}

func (r *smallerResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	srcInfo, err := os.Stat(args.Src)
	if err != nil {
		return "", false, nil, err
	}
	dstInfo, err := os.Stat(args.Dst)
	if err != nil {
		return "", false, nil, err
	}
	if srcInfo.Size() >= dstInfo.Size() {
		return "", false, nil, nil
	}
	finalize, err := swapAside(args.Dst)
	if err != nil {
		return "", false, nil, fmt.Errorf("smaller: failed to set aside larger destination: %w", err)
	}
	return args.Dst, true, finalize, nil
}

func (r *smallerResolver) SkipMessage() string { return "file skipped - destination is smaller" }

type hashCheckResolver struct{}

func (r *hashCheckResolver) Resolve(args ConflictArgs) (string, bool, FinalizeFunc, error) {
	match, err := compareFileHashes(args.Src, args.Dst)
	if err != nil {
		return "", false, nil, err
	}
	if match {
		if err := os.Remove(args.Src); err != nil && !os.IsNotExist(err) {
			return "", false, nil, fmt.Errorf("failed to remove duplicate source file: %w", err)
		}
		return "", false, nil, nil
	}
	path, err := getUniqueDestinationPath(args.DestDir, args.FileName)
	if err != nil {
		return "", false, nil, err
	}
	return path, true, nil, nil
}

func (r *hashCheckResolver) SkipMessage() string { return "duplicate file removed from source" }
