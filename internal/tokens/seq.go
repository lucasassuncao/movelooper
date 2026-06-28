package tokens

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// seqDirLocks serialises the per-directory scan that computes the next sequence
// number, so two resolutions for the same dir do not read the directory at once.
//
// NOTE: the lock is released as soon as the number is computed, before the file
// is written to disk, so on its own it does not guarantee unique numbers under
// concurrent moves. That is acceptable today because the move pipeline is
// single-threaded (the one-shot run processes categories serially and watch runs
// on a single ticker goroutine). Revisit and hold the lock across the whole
// resolve-and-move sequence if moves are ever parallelised.
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

var (
	leadingNumber  = regexp.MustCompile(`^(\d+)`)
	trailingNumber = regexp.MustCompile(`(\d+)$`)
	seqToken       = regexp.MustCompile(`\{seq(?::(\d+))?\}`)
)

// seqPos indicates where a {seq} token sits in the rename template, which
// controls where resolveSeqAt looks for an existing number in candidate files:
// a number at the start of the name (e.g. "001_photo") or at the end ("photo_001").
type seqPos int

const (
	seqLeading seqPos = iota
	seqTrailing
)

// seqTokenPosition reports whether the token spanning loc sits at the very end of
// template (seqTrailing) or anywhere else (seqLeading). A token that starts at
// index 0 — including a bare "{seq}" that is both first and last — is treated as
// leading, which scans correctly since the number then spans the whole base name.
func seqTokenPosition(template string, loc []int) seqPos {
	if loc[0] == 0 {
		return seqLeading
	}
	if loc[1] == len(template) {
		return seqTrailing
	}
	return seqLeading
}

// ResolveSeq scans destDir for files whose names begin with a decimal number,
// finds the maximum, and returns max+1. Returns 1 when the directory is empty,
// does not exist, or contains no files with a leading number.
func ResolveSeq(destDir string) int {
	return resolveSeqAt(destDir, seqLeading)
}

// resolveSeqAt scans destDir for files carrying a decimal number at the position
// indicated by pos — leading or trailing, ignoring the file extension — finds the
// maximum, and returns max+1. Returns 1 when no candidate carries a number.
func resolveSeqAt(destDir string, pos seqPos) int {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return 1
	}
	re := leadingNumber
	if pos == seqTrailing {
		re = trailingNumber
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if m := re.FindStringSubmatch(base); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

// SeqAllocator hands out sequence numbers per destination directory without
// re-scanning the directory for every file. The first request for a directory
// seeds the counter from the existing files (the same scan ResolveSeq* perform);
// subsequent requests increment in memory. This turns an O(files) directory scan
// per moved file into a single scan per directory for a whole batch.
//
// Not safe for concurrent use: the move pipeline is single-threaded (see the
// seqDirLocks note). A failed or skipped move leaves a gap in the numbering,
// which is harmless — sequence numbers are not guaranteed to be contiguous.
type SeqAllocator struct {
	dirs map[string]*seqState
}

// seqState holds the next value of each sequence kind for one destination
// directory. A field of 0 means "not yet seeded", which is unambiguous because
// every seed (resolveSeq*) returns at least 1.
type seqState struct {
	num, alpha, roman int
}

// NewSeqAllocator returns an empty allocator ready to seed directories on demand.
func NewSeqAllocator() *SeqAllocator {
	return &SeqAllocator{dirs: map[string]*seqState{}}
}

// state returns the per-directory counter, creating it on first use.
func (a *SeqAllocator) state(destDir string) *seqState {
	s := a.dirs[destDir]
	if s == nil {
		s = &seqState{}
		a.dirs[destDir] = s
	}
	return s
}

func (a *SeqAllocator) nextNum(destDir string, pos seqPos) int {
	s := a.state(destDir)
	if s.num == 0 {
		s.num = resolveSeqAt(destDir, pos)
	}
	n := s.num
	s.num++
	return n
}

func (a *SeqAllocator) nextAlpha(destDir string) int {
	s := a.state(destDir)
	if s.alpha == 0 {
		s.alpha = resolveSeqAlphaInt(destDir)
	}
	n := s.alpha
	s.alpha++
	return n
}

func (a *SeqAllocator) nextRoman(destDir string) int {
	s := a.state(destDir)
	if s.roman == 0 {
		s.roman = resolveSeqRomanInt(destDir)
	}
	n := s.roman
	s.roman++
	return n
}

func preProcessSeq(template, destDir string, alloc *SeqAllocator) string {
	loc := seqToken.FindStringIndex(template)
	if loc == nil {
		return template
	}
	pos := seqTokenPosition(template, loc)
	next := resolveSeqAt(destDir, pos)
	if alloc != nil {
		next = alloc.nextNum(destDir, pos)
	}
	return seqToken.ReplaceAllStringFunc(template, func(tok string) string {
		m := seqToken.FindStringSubmatch(tok)
		if m[1] == "" {
			return strconv.Itoa(next)
		}
		width, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("%0*d", width, next)
	})
}

var (
	leadingAlpha  = regexp.MustCompile(`^([a-z]+)`)
	seqAlphaToken = regexp.MustCompile(`\{seq-alpha\}`)
)

// ResolveSeqAlpha scans destDir for files with leading lowercase alpha prefixes
// and returns the next label in Excel-style sequence (a, b, ..., z, aa, ab, ...).
func ResolveSeqAlpha(destDir string) string {
	return intToAlpha(resolveSeqAlphaInt(destDir))
}

// resolveSeqAlphaInt is the 1-based integer behind ResolveSeqAlpha, returning 1
// (which maps to "a") when the directory is empty or unreadable.
func resolveSeqAlphaInt(destDir string) int {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return 1
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
	return max + 1
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

func preProcessSeqAlpha(template, destDir string, alloc *SeqAllocator) string {
	if !seqAlphaToken.MatchString(template) {
		return template
	}
	label := ResolveSeqAlpha(destDir)
	if alloc != nil {
		label = intToAlpha(alloc.nextAlpha(destDir))
	}
	return seqAlphaToken.ReplaceAllString(template, label)
}

// leadingRoman matches a non-empty roman numeral at the start of a filename (lowercase).
var (
	leadingRoman  = regexp.MustCompile(`^(m{0,4}(?:cm|cd|d?c{0,3})(?:xc|xl|l?x{0,3})(?:ix|iv|v?i{0,3}))(?:[^a-z]|$)`)
	seqRomanToken = regexp.MustCompile(`\{seq-roman\}`)
)

// ResolveSeqRoman scans destDir for files with leading roman numeral prefixes
// and returns the next roman numeral in sequence.
func ResolveSeqRoman(destDir string) string {
	return intToRoman(resolveSeqRomanInt(destDir))
}

// resolveSeqRomanInt is the 1-based integer behind ResolveSeqRoman, returning 1
// (which maps to "i") when the directory is empty or unreadable.
func resolveSeqRomanInt(destDir string) int {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return 1
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
	return max + 1
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

func preProcessSeqRoman(template, destDir string, alloc *SeqAllocator) string {
	if !seqRomanToken.MatchString(template) {
		return template
	}
	label := ResolveSeqRoman(destDir)
	if alloc != nil {
		label = intToRoman(alloc.nextRoman(destDir))
	}
	return seqRomanToken.ReplaceAllString(template, label)
}
