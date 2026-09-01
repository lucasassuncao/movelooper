package tokens

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// leadingSeqTemplate and trailingSeqTemplate are the two shapes a {seq} rename
// takes; the seed scan reads the destination through whichever one is in use.
const (
	leadingSeqTemplate  = "{seq}_{name}.{ext}"
	trailingSeqTemplate = "{name}_{seq}.{ext}"
	alphaSeqTemplate    = "{seq-alpha}_{name}.{ext}"
	romanSeqTemplate    = "{seq-roman}_{name}.{ext}"
)

// seqLoc returns the position of the {seq} token in template.
func seqLoc(t *testing.T, template string) []int {
	t.Helper()
	loc := seqToken.FindStringIndex(template)
	require.NotNil(t, loc)
	return loc
}

// testResolveSeq defines a structure for test cases of the leading-number seq
// resolution, containing the name of the test case, a list of existing file
// names, a flag for non-existent directory, and the expected next sequence
// number.
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
			assert.Equal(t, tt.want, resolveSeqNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
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
			assert.Equal(t, tt.want, resolveSeqNum(dir, trailingSeqTemplate, seqLoc(t, trailingSeqTemplate)))
		})
	}
}

// TestSeqScanPattern verifies the matcher built from the template: the literal
// text touching the token is required, and the token's position decides which
// anchors apply.
func TestSeqScanPattern(t *testing.T) {
	t.Parallel()
	cases := []struct {
		template string
		want     string
	}{
		{"{seq}_{name}.{ext}", `^(\d+)_`},
		{"{seq}", `^(\d+)$`},
		{"{name}_{seq}.{ext}", `_(\d+)\.`},
		{"{name}_{seq}", `_(\d+)$`},
		{"IMG_{seq}.{ext}", `^IMG_(\d+)\.`},
	}
	for _, c := range cases {
		t.Run(c.template, func(t *testing.T) {
			t.Parallel()
			re := seqScanPattern(c.template, seqLoc(t, c.template), numValuePattern)
			require.NotNil(t, re)
			assert.Equal(t, c.want, re.String())
		})
	}
}

// TestSeqSeedIgnoresForeignNames is the regression test for sequence seeds read
// off filenames the template never produced. Every filename is a run of letters
// and plenty are valid roman numerals, so without the template's shape the
// first file of a batch was labelled from whatever happened to sit in the
// destination: "vacation.jpg" made {seq-alpha} start at "vacatioo".
func TestSeqSeedIgnoresForeignNames(t *testing.T) {
	t.Parallel()

	t.Run("alpha ignores ordinary filenames", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		for _, f := range []string{"vacation.jpg", "report.pdf"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
		}
		assert.Equal(t, "a", ResolveSeqAlpha(dir, alphaSeqTemplate))
	})

	t.Run("roman ignores words spelled with roman letters", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		for _, f := range []string{"mix.png", "civic.jpg", "dim.txt"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
		}
		assert.Equal(t, "i", ResolveSeqRoman(dir, romanSeqTemplate))
	})

	t.Run("numeric ignores a number that is not where the template puts it", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "report_2026.pdf"), []byte("x"), 0o644))
		assert.Equal(t, 1, resolveSeqNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
	})

	t.Run("numeric still resumes from a name the template did produce", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "0007_report.pdf"), []byte("x"), 0o644))
		assert.Equal(t, 8, resolveSeqNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
	})
}

// TestSeqAllocator verifies the per-batch counter seeds from existing files once
// and then increments in memory, keeping a separate counter per directory.
func TestSeqAllocator(t *testing.T) {
	t.Parallel()

	t.Run("numeric seeds from existing max then increments", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "0002_seed.jpg"), []byte("x"), 0o644))
		a := NewSeqAllocator()
		assert.Equal(t, 3, a.nextNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
		assert.Equal(t, 4, a.nextNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
		assert.Equal(t, 5, a.nextNum(dir, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
	})

	t.Run("separate directories keep independent counters", func(t *testing.T) {
		t.Parallel()
		a := NewSeqAllocator()
		d1, d2 := t.TempDir(), t.TempDir()
		assert.Equal(t, 1, a.nextNum(d1, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
		assert.Equal(t, 1, a.nextNum(d2, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
		assert.Equal(t, 2, a.nextNum(d1, leadingSeqTemplate, seqLoc(t, leadingSeqTemplate)))
	})

	t.Run("alpha and roman seed then increment", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		a := NewSeqAllocator()
		assert.Equal(t, "a", intToAlpha(a.nextAlpha(dir, alphaSeqTemplate)))
		assert.Equal(t, "b", intToAlpha(a.nextAlpha(dir, alphaSeqTemplate)))
		assert.Equal(t, "i", intToRoman(a.nextRoman(dir, romanSeqTemplate)))
		assert.Equal(t, "ii", intToRoman(a.nextRoman(dir, romanSeqTemplate)))
	})
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
			assert.Equal(t, tt.want, ResolveSeqAlpha(dir, alphaSeqTemplate))
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
			assert.Equal(t, tt.want, ResolveSeqRoman(dir, romanSeqTemplate))
		})
	}
}

// TestPreProcessSeqScansOnce is the regression test for the allocator being
// bypassed: preProcessSeq used to compute the seed by scanning the directory
// and then overwrite it with the allocator's value, paying for a full scan on
// every file. A file appearing mid-batch proves whether the scan still happens:
// with the allocator, the counter keeps counting from its seed and never sees it.
func TestPreProcessSeqScansOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	alloc := NewSeqAllocator()

	assert.Equal(t, "1_{name}.{ext}", preProcessSeq(leadingSeqTemplate, dir, alloc))

	// Something else drops a high-numbered file into the destination.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0900_other.jpg"), []byte("x"), 0o644))

	assert.Equal(t, "2_{name}.{ext}", preProcessSeq(leadingSeqTemplate, dir, alloc),
		"the allocator must keep counting in memory, not re-scan the directory")
}

// TestPreProcessSeqWithoutAllocatorReadsDisk keeps the other half honest: with no
// allocator (the single-file path) the seed does come from the directory.
func TestPreProcessSeqWithoutAllocatorReadsDisk(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0004_x.jpg"), []byte("x"), 0o644))
	assert.Equal(t, "5_{name}.{ext}", preProcessSeq(leadingSeqTemplate, dir, nil))
}
