package helper

import (
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
	var ext = "." + extension

	for _, file := range files {
		if file.Type().IsRegular() {
			if strings.HasSuffix(file.Name(), strings.ToUpper(ext)) || strings.HasSuffix(file.Name(), strings.ToLower(ext)) {
				count++
			}
		}
	}

	return count
}

// MoveFiles moves files with the specified extension from the source directory to the destination directory.
// The destination path includes a subdirectory named after the extension.
func MoveFiles(m *models.Movelooper, category *models.MediaConfig, files []os.DirEntry, extension string) {
	var ext = "." + extension

	for _, file := range files {
		sourceFile := filepath.Join(category.Source, file.Name())
		destinationFile := filepath.Join(category.Destination, extension, file.Name())

		if strings.HasSuffix(file.Name(), strings.ToUpper(ext)) || strings.HasSuffix(file.Name(), strings.ToLower(ext)) {
			_, err := os.Stat(destinationFile)
			if err == nil {
				m.Logger.Warn("destination file already exists",
					m.Logger.Args("file", destinationFile))
			}

			err = os.Rename(sourceFile, destinationFile)
			if err != nil {
				m.Logger.Error("failed to move file", m.Logger.Args("file", sourceFile), m.Logger.Args("error", err.Error()))
			}

			m.Logger.Info("successfully moved file", m.Logger.Args("source", sourceFile), m.Logger.Args("destination", destinationFile))
		}
	}
}
