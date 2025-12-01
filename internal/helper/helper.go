// Package helper provides utility functions for file and directory operations.
package helper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lucasassuncao/movelooper/internal/models"
)

// CreateDirectory checks if the specified directory exists, and if not, creates it with full permissions.
func CreateDirectory(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// ReadDirectory reads the contents of a given directory and returns the files.
func ReadDirectory(path string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// ValidateFiles checks each file in the provided list to see if it is a regular file
// and has the specified extension (case-insensitive). It returns the count of matching files.
func ValidateFiles(files []os.DirEntry, extension string) int {
	var count int

	for _, file := range files {
		if file.Type().IsRegular() && HasExtension(file, extension) {
			count++
		}
	}

	return count
}

// MoveFiles moves files with the specified extension from the source directory to the destination directory.
// The destination path includes a subdirectory named after the extension, avoiding overwriting files.
func MoveFiles(m *models.Movelooper, category *models.Category, files []os.DirEntry, extension string) {
	for _, file := range files {
		if !HasExtension(file, extension) {
			continue
		}

		sourcePath := filepath.Join(category.Source, file.Name())

		var destDir string
		if category.UseExtensionSubfolder {
			destDir = filepath.Join(category.Destination, extension)
		} else {
			destDir = category.Destination
		}

		destPath := filepath.Join(destDir, file.Name())
		strategy := category.ConflictStrategy
		if strategy == "" {
			strategy = "rename"
		}

		// Check if file already exists in destination
		if _, err := os.Stat(destPath); err == nil {
			resolvedPath, shouldMove, err := resolveConflict(strategy, sourcePath, destPath, destDir, file.Name())
			if err != nil {
				m.Logger.Error("error to solve conflicts", m.Logger.Args("file", file.Name()), m.Logger.Args("error", err.Error()))
				continue
			}

			if !shouldMove {
				if strategy == "skip" {
					m.Logger.Info("file skipped due to conflict strategy", m.Logger.Args("file", file.Name()))
				}

				if strategy == "hash_check" {
					m.Logger.Info("file identical, source removed", m.Logger.Args("file", file.Name()))
				}
				continue
			}
			destPath = resolvedPath
		}

		err := moveFile(sourcePath, destPath)
		if err != nil {
			m.Logger.Error("failed to move file", m.Logger.Args("file", sourcePath), m.Logger.Args("error", err.Error()))
			continue
		}

		m.Logger.Info("successfully moved file", m.Logger.Args("source", sourcePath), m.Logger.Args("destination", destPath))
	}
}

// resolveConflict handles file name conflicts based on the specified strategy.
func resolveConflict(strategy, src, dst, destDir, fileName string) (string, bool, error) {
	switch strategy {
	case "overwrite":
		// Removes the destination file to allow overwrite
		if err := os.Remove(dst); err != nil {
			return "", false, fmt.Errorf("failed to remove destination file for overwrite: %w", err)
		}
		return dst, true, nil

	case "skip":
		return "", false, nil

	case "hash_check":
		match, err := compareFileHashes(src, dst)
		if err != nil {
			return "", false, err
		}
		if match {
			if err := os.Remove(src); err != nil {
				return "", false, fmt.Errorf("failed to remove duplicate source file: %w", err)
			}
			return "", false, nil
		}
		// If contents are different but names are the same, fall through to default (rename)
		fallthrough

	case "rename":
		fallthrough
	default:
		return getUniqueDestinationPath(destDir, fileName), true, nil
	}
}

// compareFileHashes compares the SHA-256 hashes of two files to determine if they are identical
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

// calculateHash computes the SHA-256 hash of a file's contents
func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
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

// moveFile attempts to move a file from source to destination
func moveFile(src, dst string) error {
	return os.Rename(src, dst)
}

// getUniqueDestinationPath ensures no file is overwritten by appending (n) if needed
func getUniqueDestinationPath(destDir, fileName string) string {
	ext := filepath.Ext(fileName)
	nameOnly := strings.TrimSuffix(fileName, ext)

	destPath := filepath.Join(destDir, fileName)
	counter := 1

	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		newName := fmt.Sprintf("%s(%d)%s", nameOnly, counter, ext)
		destPath = filepath.Join(destDir, newName)
		counter++
	}

	return destPath
}

// HasExtension checks if a file has a given extension (case-insensitive)
func HasExtension(file os.DirEntry, extension string) bool {
	ext := "." + extension
	fileExt := strings.ToLower(filepath.Ext(file.Name()))
	return fileExt == strings.ToLower(ext)
}

// GenerateLogArgs generates log arguments for a given extension.
func GenerateLogArgs(files []os.DirEntry, extension string) []interface{} {
	var logArgs []interface{}
	for _, file := range files {
		if HasExtension(file, extension) {
			logArgs = append(logArgs, "name", file.Name())
		}
	}
	return logArgs
}

// MatchesRegex checks if the file name matches the provided regex pattern
func MatchesRegex(fileName, pattern string) bool {
	matched, err := regexp.MatchString(pattern, fileName)
	if err != nil {
		return false
	}
	return matched
}
