package tokens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPreProcessHash defines the structure for test cases of the preProcessHash function,
// containing the template string, expected output, and a flag to simulate an unreadable file.
type testPreProcessHash struct {
	name     string
	template string
	want     string
	badPath  bool
}

// testPreProcessHashTestCases defines a set of test cases for the preProcessHash function,
// covering md5, sha256, combined usage, passthrough, and unreadable file scenarios.
var testPreProcessHashTestCases = []testPreProcessHash{
	// MD5 of "hello" = 5d41402abc4b2a76b9719d911017c592
	{"md5 default 8 chars", "{md5}_{name}.{ext}", "5d41402a_{name}.{ext}", false},
	{"md5:N custom length", "{md5:4}_{name}", "5d41_{name}", false},
	// SHA-256 of "hello" = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	{"sha256:N", "{sha256:6}_{name}", "2cf24d_{name}", false},
	// both md5 and sha256 in same template
	{"md5 and sha256 combined", "{md5:4}_{sha256:4}_{name}", "5d41_2cf2_{name}", false},
	{"no hash token passthrough", "{name}.{ext}", "{name}.{ext}", false},
	{"unreadable file returns unknown", "{md5}_{name}", "unknown_{name}", true},
	{"sha256 unreadable file truncates unknown", "{sha256:4}_{name}", "unkn_{name}", true},
}

// TestPreProcessHash tests the preProcessHash function with various hash tokens and file scenarios.
func TestPreProcessHash(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	for _, tt := range testPreProcessHashTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := path
			if tt.badPath {
				src = "/nonexistent/file.txt"
			}
			got := preProcessHash(tt.template, src)
			assert.Equal(t, tt.want, got)
		})
	}
}
