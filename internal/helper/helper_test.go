package helper

import (
	"io/fs"
	"os"
	"testing"
	"time"
)

// MockDirEntry implements os.DirEntry for testing purposes
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() fs.FileMode          { return 0 }
func (m MockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// MockFileInfo implements fs.FileInfo for testing purposes (needed for Type().IsRegular())
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return 0 }
func (m MockFileInfo) Mode() fs.FileMode  { return 0 }
func (m MockFileInfo) ModTime() time.Time { return time.Now() }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() any           { return nil }

// We need a way to mock Type().IsRegular().
// os.DirEntry.Type() returns fs.FileMode.
// fs.FileMode.IsRegular() returns true if it's a regular file.
// So we need to return a FileMode that represents a regular file.

type MockDirEntryRegular struct {
	name string
}

func (m MockDirEntryRegular) Name() string               { return m.name }
func (m MockDirEntryRegular) IsDir() bool                { return false }
func (m MockDirEntryRegular) Type() fs.FileMode          { return 0 } // 0 is a regular file
func (m MockDirEntryRegular) Info() (fs.FileInfo, error) { return nil, nil }

type MockDirEntryDir struct {
	name string
}

func (m MockDirEntryDir) Name() string               { return m.name }
func (m MockDirEntryDir) IsDir() bool                { return true }
func (m MockDirEntryDir) Type() fs.FileMode          { return fs.ModeDir }
func (m MockDirEntryDir) Info() (fs.FileInfo, error) { return nil, nil }

func TestHasExtension(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		extension string
		want      bool
	}{
		{"Exact match", "image.jpg", "jpg", true},
		{"Case insensitive file", "image.JPG", "jpg", true},
		{"Case insensitive ext", "image.jpg", "JPG", true},
		{"No match", "image.png", "jpg", false},
		{"No extension", "image", "jpg", false},
		{"Dot in filename", "my.image.jpg", "jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := MockDirEntryRegular{name: tt.filename}
			if got := HasExtension(entry, tt.extension); got != tt.want {
				t.Errorf("HasExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateFiles(t *testing.T) {
	files := []os.DirEntry{
		MockDirEntryRegular{name: "test1.jpg"},
		MockDirEntryRegular{name: "test2.png"},
		MockDirEntryRegular{name: "test3.JPG"},
		MockDirEntryDir{name: "folder.jpg"}, // Should be ignored because it's a dir
	}

	tests := []struct {
		name      string
		extension string
		want      int
	}{
		{"Count jpg", "jpg", 2},
		{"Count png", "png", 1},
		{"Count gif", "gif", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateFiles(files, tt.extension); got != tt.want {
				t.Errorf("ValidateFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateLogArgs(t *testing.T) {
	files := []os.DirEntry{
		MockDirEntryRegular{name: "test1.jpg"},
		MockDirEntryRegular{name: "test2.png"},
	}

	args := GenerateLogArgs(files, "jpg")
	if len(args) != 2 { // name, test1.jpg
		t.Errorf("GenerateLogArgs() length = %v, want 2", len(args))
	}
	if args[0] != "name" || args[1] != "test1.jpg" {
		t.Errorf("GenerateLogArgs() content = %v", args)
	}
}
