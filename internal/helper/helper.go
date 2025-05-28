package helper

import (
	"fmt"
	"movelooper/internal/models"
	"os"
	"path/filepath"
	"strings"
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
func MoveFiles(m *models.Movelooper, category *models.CategoryConfig, files []os.DirEntry, extension string) {
	for _, file := range files {
		if !HasExtension(file, extension) {
			continue
		}

		sourcePath := filepath.Join(category.Source, file.Name())
		destDir := filepath.Join(category.Destination, extension)

		finalDestPath := getUniqueDestinationPath(destDir, file.Name())

		err := moveFile(sourcePath, finalDestPath)
		if err != nil {
			m.Logger.Error("failed to move file", m.Logger.Args("file", sourcePath), m.Logger.Args("error", err.Error()))
			continue
		}

		m.Logger.Info("successfully moved file", m.Logger.Args("source", sourcePath), m.Logger.Args("destination", finalDestPath))
	}
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
