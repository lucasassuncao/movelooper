package cmd

import (
	"movelooper/models"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// getCategories retrieves the category names from the Viper configuration and returns them as a slice.
func getCategories(v *viper.Viper) []string {
	var categories = make([]string, 0)

	for key := range v.GetStringMap("categories") {
		categories = append(categories, key)
	}

	return categories
}

// createDirectory checks if the specified directory exists, and if not, creates it with full permissions.
func createDirectory(m *models.Movelooper, dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			m.Logger.Error("failed to create directory", m.Logger.Args("error", err.Error()))
			return
		}
		m.Logger.Info("successfully created directory", m.Logger.Args("directory", dir))
		return
	}
}

// readDirectory reads the contents of a given directory and returns the files.
func readDirectory(m *models.Movelooper, path string) []os.DirEntry {
	files, err := os.ReadDir(path)
	if err != nil {
		m.Logger.Error("failed to read directory",
			m.Logger.Args("path", path),
			m.Logger.Args("error", err.Error()),
		)
		return nil
	}

	return files
}

// validateFiles checks each file in the provided list to see if it is a regular file
// and has the specified extension (case-insensitive). It returns the count of matching files.
func validateFiles(files []os.DirEntry, extension string) int {
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

// moveFile moves files with the specified extension from the source directory to the destination directory.
// The destination path includes a subdirectory named after the extension.
func moveFile(m *models.Movelooper, files []os.DirEntry, extension string) {
	var ext = "." + extension

	for _, file := range files {
		sourceFile := filepath.Join(m.MediaConfig.Source, file.Name())
		destinationFile := filepath.Join(m.MediaConfig.Destination, extension, file.Name())

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
