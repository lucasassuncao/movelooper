package tokens

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testResolveSeq defines a structure for test cases of the ResolveSeq function,
// containing the name of the test case, a list of existing file names, a flag for non-existent directory, and the expected next sequence number.
type testResolveSeq struct {
	name     string
	existing []string
	nonExist bool
	want     int
}

// testResolveSeqTestCases defines a set of test cases for the ResolveSeq function,
// covering various scenarios of existing files and directory states.
var testResolveSeqTestCases = []testResolveSeq{
	{"empty directory returns 1", nil, false, 1},
	{"directory does not exist returns 1", nil, true, 1},
	{"single file with leading number", []string{"0001_photo.jpg"}, false, 2},
	{"multiple files picks max", []string{"0001_a.jpg", "0005_b.jpg", "0003_c.jpg"}, false, 6},
	{"files without leading number are ignored", []string{"photo.jpg", "banner.png"}, false, 1},
	{"mixed: some with numbers some without", []string{"0002_x.jpg", "logo.png"}, false, 3},
}

// TestResolveSeq tests the ResolveSeq function with various scenarios of existing files and directory states
// to ensure it correctly identifies the next sequence number.
func TestResolveSeq(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolveSeqTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dir string
			if tt.nonExist {
				dir = filepath.Join(t.TempDir(), "nonexistent")
			} else {
				dir = t.TempDir()
				for _, name := range tt.existing {
					require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
				}
			}
			assert.Equal(t, tt.want, ResolveSeq(dir))
		})
	}
}

// testResolveSeqTrailingTestCases covers numbers at the END of the base name,
// which is how "{name}_{seq}" templates lay out their sequence.
var testResolveSeqTrailingTestCases = []testResolveSeq{
	{"empty directory returns 1", nil, false, 1},
	{"single file with trailing number", []string{"photo_0001.jpg"}, false, 2},
	{"multiple files picks max", []string{"a_0001.jpg", "b_0005.jpg", "c_0003.jpg"}, false, 6},
	{"leading numbers are ignored", []string{"0009_photo.jpg"}, false, 1},
	{"files without trailing number are ignored", []string{"photo.jpg", "banner.png"}, false, 1},
}

// TestResolveSeqTrailing ensures resolveSeqAt finds the next number when the
// sequence sits at the end of the filename (e.g. "{name}_{seq}").
func TestResolveSeqTrailing(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolveSeqTrailingTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, name := range tt.existing {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
			}
			assert.Equal(t, tt.want, resolveSeqAt(dir, seqTrailing))
		})
	}
}

// TestSeqTokenPosition verifies the template-position detection that decides
// whether {seq} scans leading or trailing numbers.
func TestSeqTokenPosition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		template string
		want     seqPos
	}{
		{"{seq}_{name}", seqLeading},
		{"{seq}", seqLeading},
		{"{name}_{seq}", seqTrailing},
		{"{name}_{seq}_{ext}", seqLeading}, // token in the middle defaults to leading
	}
	for _, c := range cases {
		t.Run(c.template, func(t *testing.T) {
			t.Parallel()
			loc := seqToken.FindStringIndex(c.template)
			require.NotNil(t, loc)
			assert.Equal(t, c.want, seqTokenPosition(c.template, loc))
		})
	}
}

type testAlphaConversion struct {
	n    int
	want string
}

var testAlphaConversionTestCases = []testAlphaConversion{
	{1, "a"}, {2, "b"}, {26, "z"},
	{27, "aa"}, {28, "ab"}, {52, "az"},
	{53, "ba"}, {702, "zz"}, {703, "aaa"},
}

func TestAlphaConversion(t *testing.T) {
	t.Parallel()
	for _, c := range testAlphaConversionTestCases {
		t.Run(fmt.Sprintf("%d=%s", c.n, c.want), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, intToAlpha(c.n))
			assert.Equal(t, c.n, alphaToInt(c.want))
		})
	}
}

type testResolveSeqAlpha struct {
	name     string
	existing []string
	nonExist bool
	want     string
}

var testResolveSeqAlphaTestCases = []testResolveSeqAlpha{
	{"empty dir returns a", nil, false, "a"},
	{"nonexistent dir returns a", nil, true, "a"},
	{"after a returns b", []string{"a_doc.pdf"}, false, "b"},
	{"after z returns aa", []string{"z_doc.pdf"}, false, "aa"},
	{"after aa returns ab", []string{"aa_doc.pdf"}, false, "ab"},
	{"picks max", []string{"a_x.pdf", "c_x.pdf", "b_x.pdf"}, false, "d"},
	{"ignores non-alpha prefix", []string{"1_x.pdf"}, false, "a"},
}

func TestResolveSeqAlpha(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolveSeqAlphaTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dir string
			if tt.nonExist {
				dir = filepath.Join(t.TempDir(), "nonexistent")
			} else {
				dir = t.TempDir()
				for _, f := range tt.existing {
					require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
				}
			}
			assert.Equal(t, tt.want, ResolveSeqAlpha(dir))
		})
	}
}

type testRomanConversion struct {
	n    int
	want string
}

var testRomanConversionTestCases = []testRomanConversion{
	{1, "i"}, {4, "iv"}, {5, "v"}, {9, "ix"},
	{10, "x"}, {14, "xiv"}, {40, "xl"}, {90, "xc"},
	{100, "c"}, {400, "cd"}, {500, "d"}, {900, "cm"},
	{1000, "m"}, {1999, "mcmxcix"}, {2024, "mmxxiv"},
}

func TestRomanConversion(t *testing.T) {
	t.Parallel()
	for _, c := range testRomanConversionTestCases {
		t.Run(fmt.Sprintf("%d=%s", c.n, c.want), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, intToRoman(c.n))
			assert.Equal(t, c.n, romanToInt(c.want))
		})
	}
}

type testResolveSeqRoman struct {
	name     string
	existing []string
	nonExist bool
	want     string
}

var testResolveSeqRomanTestCases = []testResolveSeqRoman{
	{"empty dir returns i", nil, false, "i"},
	{"nonexistent dir returns i", nil, true, "i"},
	{"after i returns ii", []string{"i_doc.pdf"}, false, "ii"},
	{"after iv returns v", []string{"iv_doc.pdf"}, false, "v"},
	{"picks max", []string{"i_x.pdf", "iii_x.pdf", "ii_x.pdf"}, false, "iv"},
	{"ignores non-roman prefix", []string{"1_x.pdf", "a_x.pdf"}, false, "i"},
}

func TestResolveSeqRoman(t *testing.T) {
	t.Parallel()
	for _, tt := range testResolveSeqRomanTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dir string
			if tt.nonExist {
				dir = filepath.Join(t.TempDir(), "nonexistent")
			} else {
				dir = t.TempDir()
				for _, f := range tt.existing {
					require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
				}
			}
			assert.Equal(t, tt.want, ResolveSeqRoman(dir))
		})
	}
}
