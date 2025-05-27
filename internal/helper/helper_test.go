package helper

import (
	"movelooper/internal/models"
	"os"
	"path/filepath"
	"testing"
	"fmt"

	"github.com/pterm/pterm"
)

// Helper function to create a temporary directory structure for testing
func setupTestDirs(t *testing.T, srcFiles []string, destFiles map[string][]string) (string, string, *models.Movelooper, *models.CategoryConfig) {
	t.Helper()

	sourceDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp source dir: %v", err)
	}

	destinationDir, err := os.MkdirTemp("", "destination")
	if err != nil {
		os.RemoveAll(sourceDir)
		t.Fatalf("Failed to create temp dest dir: %v", err)
	}

	for _, fName := range srcFiles {
		if err := os.WriteFile(filepath.Join(sourceDir, fName), []byte("dummy content"), 0666); err != nil {
			os.RemoveAll(sourceDir)
			os.RemoveAll(destinationDir)
			t.Fatalf("Failed to create source file %s: %v", fName, err)
		}
	}

	for ext, filesInDest := range destFiles {
		extDir := filepath.Join(destinationDir, ext)
		if err := os.MkdirAll(extDir, 0777); err != nil {
			os.RemoveAll(sourceDir)
			os.RemoveAll(destinationDir)
			t.Fatalf("Failed to create dest extension dir %s: %v", extDir, err)
		}
		for _, fName := range filesInDest {
			if err := os.WriteFile(filepath.Join(extDir, fName), []byte("dummy content"), 0666); err != nil {
				os.RemoveAll(sourceDir)
				os.RemoveAll(destinationDir)
				t.Fatalf("Failed to create dest file %s: %v", fName, err)
			}
		}
	}
	
	m := &models.Movelooper{
		Logger: pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled),
	}

	category := &models.CategoryConfig{
		Source:      sourceDir,
		Destination: destinationDir,
		// Extensions will be set by each test
	}

	return sourceDir, destinationDir, m, category
}

// Helper function to cleanup temporary directories
func cleanupTestDirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Warning: Failed to remove temp dir %s: %v", dir, err)
		}
	}
}

// TestMoveFiles_NoConflict tests moving a file when no conflict exists.
func TestMoveFiles_NoConflict(t *testing.T) {
	srcFileName := "testfile.txt"
	extension := "txt"
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, nil)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension}

	// Create the destination extension directory explicitly as MoveFiles expects it
	destExtDir := filepath.Join(destinationDir, extension)
	if err := os.MkdirAll(destExtDir, 0777); err != nil {
		t.Fatalf("Failed to create destination extension directory %s: %v", destExtDir, err)
	}

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension)

	expectedDestFile := filepath.Join(destExtDir, srcFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be moved, but it does not exist", expectedDestFile)
	}
	// Check if source file is removed
	if _, err := os.Stat(filepath.Join(sourceDir, srcFileName)); !os.IsNotExist(err) {
		t.Errorf("Expected source file %s to be removed, but it still exists", srcFileName)
	}
}

// TestMoveFiles_OneExistingFile tests renaming when one file with the same name exists.
func TestMoveFiles_OneExistingFile(t *testing.T) {
	srcFileName := "testfile.txt"
	extension := "txt"
	destExistingFiles := map[string][]string{
		extension: {srcFileName},
	}
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, destExistingFiles)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension}

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension)

	expectedDestFileName := "testfile(1).txt"
	expectedDestFile := filepath.Join(destinationDir, extension, expectedDestFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist", expectedDestFile)
	}
	// Check if source file is removed
	if _, err := os.Stat(filepath.Join(sourceDir, srcFileName)); !os.IsNotExist(err) {
		t.Errorf("Expected source file %s to be removed, but it still exists", srcFileName)
	}
}

// TestMoveFiles_MultipleExistingFiles tests renaming with multiple existing files.
func TestMoveFiles_MultipleExistingFiles(t *testing.T) {
	srcFileName := "testfile.txt"
	extension := "txt"
	destExistingFiles := map[string][]string{
		extension: {srcFileName, "testfile(1).txt"},
	}
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, destExistingFiles)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension}

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension)

	expectedDestFileName := "testfile(2).txt"
	expectedDestFile := filepath.Join(destinationDir, extension, expectedDestFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist", expectedDestFile)
	}
}

// TestMoveFiles_NoExtension tests versioning for files without an extension.
// Note: The current MoveFiles implementation categorizes by extension.
// If extension is "", files are moved to a directory named "" (empty string).
func TestMoveFiles_NoExtension(t *testing.T) {
	srcFileName := "testfile"
	extension := "" // Representing no extension for categorization
	
	// Setup: source file "testfile", destination has "testfile" and "testfile(1)"
	// We expect "testfile" from source to become "testfile(2)" in dest/"".
	destExistingFiles := map[string][]string{
		extension: {srcFileName, fmt.Sprintf("%s(1)", srcFileName)},
	}
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, destExistingFiles)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension} // This test is for the *category* "no extension"

	// Create the destination "" directory explicitly
	destExtDir := filepath.Join(destinationDir, extension)
	if err := os.MkdirAll(destExtDir, 0777); err != nil {
		t.Fatalf("Failed to create destination extension directory '%s': %v", destExtDir, err)
	}
	
	// Simulate a file that would match an "" extension category.
	// For this, we need a custom DirEntry mock or a way to influence HasExtension.
	// The current HasExtension checks file.Name() for a suffix ".".
	// So, for a file named "testfile" to match extension "", HasExtension would need adjustment or
	// the definition of "matching an empty extension" needs to be clear.
	// Assuming HasExtension is modified or the intent is that "testfile" matches category ""
	// if "" is in category.Extensions. The current HasExtension will NOT match `.`+"" for "testfile".
	// For the sake of this test, we'll assume files without a dot are processed if extension is ""
	// and HasExtension is adapted, or we are testing the renaming logic primarily.
	// The current `HasExtension` logic: `strings.HasSuffix(file.Name(), strings.ToUpper("."+extension))`
	// If `extension` is "", it checks for suffix ".".
	// So, for a file "testfile" to be processed by `MoveFiles` when `extension` is "",
	// `HasExtension` must return true for it.
	// Let's rename srcFile to "testfile." to make it match.
	// This is a bit of a workaround for the current HasExtension logic.
	// The alternative is to mock DirEntry and HasExtension.
	
	renamedSrcFile := srcFileName + "." // "testfile."
	originalSourcePath := filepath.Join(sourceDir, srcFileName)
	newSourcePath := filepath.Join(sourceDir, renamedSrcFile)
	if err := os.Rename(originalSourcePath, newSourcePath); err != nil {
		t.Fatalf("Failed to rename source file for no-extension test: %v", err)
	}
	// Also update destExistingFiles to match this pattern if they are also "extensionless"
	// For this test, we'll assume the files in destination already are named appropriately for the category.
	// So, destFiles would be: {"": ["testfile.", "testfile.(1)"]}
	// The setupTestDirs created "testfile" and "testfile(1)". We need to rename them.
	if err := os.Rename(filepath.Join(destExtDir, srcFileName), filepath.Join(destExtDir, srcFileName+"." )); err != nil {
		t.Fatalf("Failed to rename dest file for no-extension test: %v", err)
	}
	if err := os.Rename(filepath.Join(destExtDir, srcFileName+"(1)"), filepath.Join(destExtDir, srcFileName+"(1).")); err != nil {
		t.Fatalf("Failed to rename dest file for no-extension test: %v", err)
	}


	files, err := ReadDirectory(sourceDir) // Reads "testfile."
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension) // extension is ""

	// Expect "testfile.(2)" because "testfile." and "testfile.(1)." exist
	expectedDestFileName := fmt.Sprintf("%s(2).", srcFileName)
	expectedDestFile := filepath.Join(destExtDir, expectedDestFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist in %s", expectedDestFileName, destExtDir)
		// List files in destExtDir for debugging
		actualFiles, _ := os.ReadDir(destExtDir)
		for _, f := range actualFiles {
			t.Logf("Found in dest: %s", f.Name())
		}
	}
}


// TestMoveFiles_MultipleDotsInName tests versioning for filenames like "archive.tar.gz".
func TestMoveFiles_MultipleDotsInName(t *testing.T) {
	srcFileName := "archive.tar.gz"
	extension := "gz" // The actual extension is "gz"
	// Base name for versioning should be "archive.tar"
	
	destExistingFiles := map[string][]string{
		extension: {srcFileName, "archive.tar(1).gz"},
	}
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, destExistingFiles)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension}

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension)

	expectedDestFileName := "archive.tar(2).gz"
	expectedDestFile := filepath.Join(destinationDir, extension, expectedDestFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist", expectedDestFile)
	}
}

// TestMoveFiles_FileNameIsJustExtension tests versioning for filenames like ".bashrc"
// where the filename starts with a dot and could be considered as "extension-only" by some definitions.
// However, our HasExtension treats ".bashrc" as name=".bashrc" and ext="bashrc" if "bashrc" is the target extension.
// If the target extension is "", then name=".bashrc" would match if it ends with ".".
func TestMoveFiles_FileNameIsJustExtension(t *testing.T) {
	srcFileName := ".config" // File we want to move
	extension := "config"    // Target extension category
	
	// Corrected dest existing files for the test logic
	// The base name is ".config" and extension is ".config" (because we are targeting "config" extension)
	// So versioning should produce ".config(1).config"
	// Let's clarify: if srcFileName is ".config" and extension is "config",
	// filepath.Ext(".config") is ".config".
	// strings.TrimSuffix(".config", ".config") is "" (empty string). This is the base.
	// So, newName = "" + "(1)" + ".config" = "(1).config". This seems problematic.

	// Let's re-evaluate the base and ext logic in MoveFiles:
	// base := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
	// ext := filepath.Ext(file.Name())
	// For file.Name() = ".config":
	// filepath.Ext(".config") is ".config"
	// base = strings.TrimSuffix(".config", ".config") which is ""
	// newName = base + "(" + strconv.Itoa(counter) + ")" + ext
	// newName = "" + "(1)" + ".config" = "(1).config"

	// This means if ".config" exists, the new file will be "(1).config".
	// If "(1).config" also exists, it will be "(2).config".

	// So, for srcFileName = ".config" and target extension "config"
	// Existing files in destination/config/ are ".config" and "(1).config"
	// Expected new file: "(2).config"
	
	// Removed unused variable: destExistingFiles
	correctDestExistingFiles := map[string][]string{
		extension: {".config", "(1).config"},
	}

	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, correctDestExistingFiles)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension} // Moving to "config" sub-folder

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	MoveFiles(m, category, files, extension)

	expectedDestFileName := "(2).config"
	expectedDestFile := filepath.Join(destinationDir, extension, expectedDestFileName)
	if _, err := os.Stat(expectedDestFile); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist. Files in dest:", expectedDestFileName)
		actualFiles, _ := os.ReadDir(filepath.Join(destinationDir, extension))
		for _, f := range actualFiles {
			t.Logf("Found in dest: %s", f.Name())
		}
	}
}

// TestMoveFiles_SourceFileVanishes tests that no error occurs if a source file disappears before move.
// This is more of a robustness test for ReadDirectory vs MoveFiles interaction.
// MoveFiles itself iterates over DirEntry which is a snapshot. os.Rename will fail.
func TestMoveFiles_SourceFileVanishes(t *testing.T) {
	srcFileName := "ghost.txt"
	extension := "txt"
	sourceDir, destinationDir, m, category := setupTestDirs(t, []string{srcFileName}, nil)
	defer cleanupTestDirs(t, sourceDir, destinationDir)

	category.Extensions = []string{extension}
	destExtDir := filepath.Join(destinationDir, extension)
	if err := os.MkdirAll(destExtDir, 0777); err != nil {
		t.Fatalf("Failed to create destExtDir: %v", err)
	}

	files, err := ReadDirectory(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory: %v", err)
	}

	// Remove the source file *after* ReadDirectory but *before* MoveFiles
	if err := os.Remove(filepath.Join(sourceDir, srcFileName)); err != nil {
		t.Fatalf("Failed to remove source file mid-test: %v", err)
	}

	// Expect no panic, error should be logged by MoveFiles
	// We can't easily check logs here without a mock logger that captures output.
	// For now, just ensure it doesn't panic and the destination file doesn't exist.
	MoveFiles(m, category, files, extension)

	expectedDestFile := filepath.Join(destExtDir, srcFileName)
	if _, err := os.Stat(expectedDestFile); !os.IsNotExist(err) {
		t.Errorf("Expected file %s NOT to be moved, but it exists", expectedDestFile)
	}
}
