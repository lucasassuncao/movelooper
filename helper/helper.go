package helper

import (
	"movelooper/logging"
	"os"

	"github.com/logrusorgru/aurora/v4"
)

func CreateDirectory(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			logging.Logger.Error(aurora.Sprintf("Failed to create directory: %s. Error: %s", dir, err.Error()))
		}

		logging.Logger.Info(aurora.Sprintf("Successfully created directory: %s", dir))
	}
}

func MoveFileToDestination(srcFile, dstFile string) string {
	_, err := os.Stat(dstFile)
	if err == nil {
		logging.Logger.Warn(aurora.Sprintf("Destination file already exists: %s", dstFile))
		return "WARNING"
	}

	err = os.Rename(srcFile, dstFile)
	if err != nil {
		logging.Logger.Error(aurora.Sprintf("Failed to move the file: %s. Error: %s", srcFile, err.Error()))
		return "ERROR"
	}

	logging.Logger.Info(aurora.Sprintf("Successfully moved the file: %s", srcFile))
	return "SUCCESS"
}

// ByteCountDecimal converts an integer number representing a size in bytes (such as file size) into a human-readable string.
func ByteCountDecimal(b int64) string {
	const unit = 1024
	if b < unit {
		return aurora.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return aurora.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
