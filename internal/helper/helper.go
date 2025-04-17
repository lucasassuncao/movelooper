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

	for _, file := range files {
		if file.Type().IsRegular() && HasExtension(file, extension) {
			count++
		}
	}

	return count
}

// MoveFiles moves files with the specified extension from the source directory to the destination directory.
// The destination path includes a subdirectory named after the extension.
func MoveFiles(m *models.Movelooper, category *models.CategoryConfig, files []os.DirEntry, extension string) {
	for _, file := range files {
		sourceFile := filepath.Join(category.Source, file.Name())
		destinationFile := filepath.Join(category.Destination, extension, file.Name())

		if HasExtension(file, extension) {
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

func HasExtension(file os.DirEntry, extension string) bool {
	var ext = "." + extension

	return strings.HasSuffix(file.Name(), strings.ToUpper(ext)) || strings.HasSuffix(file.Name(), strings.ToLower(ext))
}

func GenerateLogArgs(files []os.DirEntry, extension string) []interface{} {
	var logArgs []interface{}
	for _, file := range files {
		if HasExtension(file, extension) {
			logArgs = append(logArgs, "name", file.Name())
		}
	}
	return logArgs
}
