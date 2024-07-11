package helper

import (
	"fmt"
	"movelooper/logging"
	"os"
)

func CreateDirectory(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			logging.Log.Errorf("Failed to create directory: %s. Error: %s.", dir, err.Error())
		}

		logging.Log.Infof("Successfully created directory: %s.", dir)
	}
}

func MoveFileToDestination(srcFile, dstFile string) string {
	err := os.Rename(srcFile, dstFile)

	if err != nil {
		logging.Log.Errorf("Failed to move the file: %s. Error: %s.", srcFile, err.Error())
		return "ERROR"
	}

	logging.Log.Infof("Successfully moved the file: %s.", srcFile)
	return "SUCCESS"
}

// Converts an integer number representing a size in bytes (such as file size) into a human-readable string.
func ByteCountDecimal(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
