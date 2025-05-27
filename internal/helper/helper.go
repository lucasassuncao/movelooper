package helper

import (
	"movelooper/internal/models"
	"os"
	"path/filepath"
	"strconv"
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
		originalDestinationFile := filepath.Join(category.Destination, extension, file.Name())
		destinationFile := originalDestinationFile

		if HasExtension(file, extension) {
			// Check if the destination file already exists
			if _, err := os.Stat(destinationFile); err == nil {
				// originalDestinationFile is the path without (n) suffix
				// destinationFile will be updated if a new name is generated
				originalPathForLog := destinationFile 
				
				// Resolve filename conflict
				base := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
				ext := filepath.Ext(file.Name())
				counter := 1
				for {
					newName := base + "(" + strconv.Itoa(counter) + ")" + ext
					potentialNewDestinationFile := filepath.Join(category.Destination, extension, newName)
					if _, errStat := os.Stat(potentialNewDestinationFile); os.IsNotExist(errStat) {
						destinationFile = potentialNewDestinationFile
						m.Logger.Info("destination file conflict, renamed to avoid overwrite",
							m.Logger.Args("original_path", originalPathForLog),
							m.Logger.Args("new_path", destinationFile))
						break
					}
					counter++
					// Safety break, in case of an unexpected loop condition, though very unlikely with file systems.
					if counter > 1000 { 
						m.Logger.Error("could not find a unique name after 1000 attempts", 
							m.Logger.Args("original_path", originalPathForLog))
						// We will try to move to the original path and likely fail, or overwrite if permissions changed.
						destinationFile = originalPathForLog 
						break
					}
				}
			}

			err := os.Rename(sourceFile, destinationFile)
			if err != nil {
				m.Logger.Error("failed to move file",
					m.Logger.Args("source", sourceFile),
					m.Logger.Args("destination", destinationFile),
					m.Logger.Args("error", err.Error()))
			} else {
				m.Logger.Info("successfully moved file",
					m.Logger.Args("source", sourceFile),
					m.Logger.Args("destination", destinationFile))
			}
		}
	}
}

// HasExtension checks if a file has a given extension (case-insensitive).
func HasExtension(file os.DirEntry, extension string) bool {
	var ext = "." + extension

	return strings.HasSuffix(file.Name(), strings.ToUpper(ext)) || strings.HasSuffix(file.Name(), strings.ToLower(ext))
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
