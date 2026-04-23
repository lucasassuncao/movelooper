package tokens

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// seqDirLocks serialises sequence-number resolution per destination directory
// to prevent two concurrent moves to the same dir from receiving identical numbers.
var seqDirLocks sync.Map

func acquireSeqLock(destDir string) func() {
	v, _ := seqDirLocks.LoadOrStore(destDir, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func hasSeqToken(template string) bool {
	return seqToken.MatchString(template) ||
		seqAlphaToken.MatchString(template) ||
		seqRomanToken.MatchString(template)
}

// --- numeric sequence ---

var (
	leadingNumber = regexp.MustCompile(`^(\d+)`)
	seqToken      = regexp.MustCompile(`\{seq(?::(\d+))?\}`)
)

// ResolveSeq scans destDir for files whose names begin with a decimal number,
// finds the maximum, and returns max+1. Returns 1 when the directory is empty,
// does not exist, or contains no files with a leading number.
func ResolveSeq(destDir string) int {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return 1
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if m := leadingNumber.FindStringSubmatch(e.Name()); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

func preProcessSeq(template string, destDir string) string {
	if !seqToken.MatchString(template) {
		return template
	}
	next := ResolveSeq(destDir)
	return seqToken.ReplaceAllStringFunc(template, func(tok string) string {
		m := seqToken.FindStringSubmatch(tok)
		if m[1] == "" {
			return strconv.Itoa(next)
		}
		width, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("%0*d", width, next)
	})
}

// --- alphabetic sequence ---

var (
	leadingAlpha  = regexp.MustCompile(`^([a-z]+)`)
	seqAlphaToken = regexp.MustCompile(`\{seq-alpha\}`)
)

// ResolveSeqAlpha scans destDir for files with leading lowercase alpha prefixes
// and returns the next label in Excel-style sequence (a, b, ..., z, aa, ab, ...).
func ResolveSeqAlpha(destDir string) string {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "a"
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if m := leadingAlpha.FindStringSubmatch(strings.ToLower(e.Name())); m != nil {
			if n := alphaToInt(m[1]); n > max {
				max = n
			}
		}
	}
	return intToAlpha(max + 1)
}

// alphaToInt converts an Excel-style column label to a 1-based integer ("a"=1, "z"=26, "aa"=27).
func alphaToInt(s string) int {
	result := 0
	for _, r := range s {
		result = result*26 + int(r-'a'+1)
	}
	return result
}

// intToAlpha converts a 1-based integer to an Excel-style column label.
func intToAlpha(n int) string {
	var b strings.Builder
	for n > 0 {
		n--
		b.WriteByte(byte('a' + n%26))
		n /= 26
	}
	rr := []rune(b.String())
	for i, j := 0, len(rr)-1; i < j; i, j = i+1, j-1 {
		rr[i], rr[j] = rr[j], rr[i]
	}
	return string(rr)
}

func preProcessSeqAlpha(template, destDir string) string {
	if !seqAlphaToken.MatchString(template) {
		return template
	}
	return seqAlphaToken.ReplaceAllString(template, ResolveSeqAlpha(destDir))
}

// --- roman numeral sequence ---

// leadingRoman matches a non-empty roman numeral at the start of a filename (lowercase).
var (
	leadingRoman  = regexp.MustCompile(`^(m{0,4}(?:cm|cd|d?c{0,3})(?:xc|xl|l?x{0,3})(?:ix|iv|v?i{0,3}))(?:[^a-z]|$)`)
	seqRomanToken = regexp.MustCompile(`\{seq-roman\}`)
)

// ResolveSeqRoman scans destDir for files with leading roman numeral prefixes
// and returns the next roman numeral in sequence.
func ResolveSeqRoman(destDir string) string {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "i"
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if m := leadingRoman.FindStringSubmatch(name); m != nil && m[1] != "" {
			if n := romanToInt(m[1]); n > max {
				max = n
			}
		}
	}
	return intToRoman(max + 1)
}

func romanToInt(s string) int {
	vals := map[byte]int{'i': 1, 'v': 5, 'x': 10, 'l': 50, 'c': 100, 'd': 500, 'm': 1000}
	result, prev := 0, 0
	for i := len(s) - 1; i >= 0; i-- {
		curr := vals[s[i]]
		if curr < prev {
			result -= curr
		} else {
			result += curr
		}
		prev = curr
	}
	return result
}

func intToRoman(n int) string {
	pairs := []struct {
		v int
		s string
	}{
		{1000, "m"}, {900, "cm"}, {500, "d"}, {400, "cd"},
		{100, "c"}, {90, "xc"}, {50, "l"}, {40, "xl"},
		{10, "x"}, {9, "ix"}, {5, "v"}, {4, "iv"}, {1, "i"},
	}
	var b strings.Builder
	for _, p := range pairs {
		for n >= p.v {
			b.WriteString(p.s)
			n -= p.v
		}
	}
	return b.String()
}

func preProcessSeqRoman(template, destDir string) string {
	if !seqRomanToken.MatchString(template) {
		return template
	}
	return seqRomanToken.ReplaceAllString(template, ResolveSeqRoman(destDir))
}
