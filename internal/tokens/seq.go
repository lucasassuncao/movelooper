package tokens

import (
	"fmt"
	"os"
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

var seqToken = regexp.MustCompile(`\{seq(?::(\d+))?\}`)

// The value each sequence kind can take, used as the capture group inside the
// matcher built from the rename template. Alpha and roman are case-insensitive
// so a label written in caps still counts.
const (
	numValuePattern   = `\d+`
	alphaValuePattern = `(?i:[a-z]+)`
	romanValuePattern = `(?i:m{0,4}(?:cm|cd|d?c{0,3})(?:xc|xl|l?x{0,3})(?:ix|iv|v?i{0,3}))`
)

// seqScanPattern builds the matcher that finds the sequence values already in
// the destination directory. It comes from the rename template, so only files
// that template could have produced are counted: the token becomes the capture
// group, and the literal text touching it on either side has to be there too.
//
// That literal is what makes the scan mean anything. Matching the value pattern
// alone reads every ordinary filename as a label: "vacation.jpg" is a run of
// lowercase letters, so {seq-alpha} continued from "vacation" instead of "a",
// and "mix.png" is a valid roman numeral, so {seq-roman} continued from 1009.
func seqScanPattern(template string, loc []int, value string) *regexp.Regexp {
	before, after := template[:loc[0]], template[loc[1]:]
	var b strings.Builder
	// Anchor whenever the literal reaches the edge of the template: with no
	// earlier token in the way, the text before the sequence is the whole start
	// of the name, and the same for the end.
	if !strings.Contains(before, "}") {
		b.WriteString(`^`)
	}
	b.WriteString(regexp.QuoteMeta(literalBefore(before)))
	b.WriteString(`(` + value + `)`)
	b.WriteString(regexp.QuoteMeta(literalAfter(after)))
	if !strings.Contains(after, "{") {
		b.WriteString(`$`)
	}
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil
	}
	return re
}

// literalBefore returns the literal text between the previous token and the one
// being resolved. Only the immediate neighbour is used: tokens resolved earlier
// in the pipeline (a hash, for instance) hold this file's own value, which no
// other file in the directory carries.
func literalBefore(s string) string {
	if i := strings.LastIndex(s, "}"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// literalAfter returns the literal text between the token being resolved and
// the next one.
func literalAfter(s string) string {
	if i := strings.Index(s, "{"); i >= 0 {
		return s[:i]
	}
	return s
}

// scanMaxSeq returns one past the highest value re captures among the files in
// destDir, or 1 when nothing matches, including an empty or unreadable
// directory. toInt converts a captured label to its ordinal.
func scanMaxSeq(destDir string, re *regexp.Regexp, toInt func(string) int) int {
	if re == nil {
		return 1
	}
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return 1
	}
	best := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if m == nil || m[1] == "" {
			continue
		}
		if n := toInt(m[1]); n > best {
			best = n
		}
	}
	return best + 1
}

// resolveSeqNum seeds {seq} from the numbers already carried by files matching
// the template's shape, and returns the next one.
func resolveSeqNum(destDir, template string, loc []int) int {
	return scanMaxSeq(destDir, seqScanPattern(template, loc, numValuePattern), func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	})
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

func (a *SeqAllocator) nextNum(destDir, template string, loc []int) int {
	s := a.state(destDir)
	if s.num == 0 {
		s.num = resolveSeqNum(destDir, template, loc)
	}
	n := s.num
	s.num++
	return n
}

func (a *SeqAllocator) nextAlpha(destDir, template string) int {
	s := a.state(destDir)
	if s.alpha == 0 {
		s.alpha = resolveSeqAlphaInt(destDir, template)
	}
	n := s.alpha
	s.alpha++
	return n
}

func (a *SeqAllocator) nextRoman(destDir, template string) int {
	s := a.state(destDir)
	if s.roman == 0 {
		s.roman = resolveSeqRomanInt(destDir, template)
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
	// The allocator seeds itself from the directory on its first call and counts
	// in memory after that. Scanning here as well would throw that scan away and
	// pay for it once per file, which is the cost the allocator exists to avoid.
	var next int
	if alloc != nil {
		next = alloc.nextNum(destDir, template, loc)
	} else {
		next = resolveSeqNum(destDir, template, loc)
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

var seqAlphaToken = regexp.MustCompile(`\{seq-alpha\}`)

// ResolveSeqAlpha returns the next Excel-style label (a, b, ..., z, aa, ab, ...)
// for destDir, reading the labels already there through the shape of template.
func ResolveSeqAlpha(destDir, template string) string {
	return intToAlpha(resolveSeqAlphaInt(destDir, template))
}

// resolveSeqAlphaInt is the 1-based integer behind ResolveSeqAlpha, returning 1
// (which maps to "a") when the directory holds no label of this shape, or is
// empty or unreadable.
func resolveSeqAlphaInt(destDir, template string) int {
	loc := seqAlphaToken.FindStringIndex(template)
	if loc == nil {
		return 1
	}
	return scanMaxSeq(destDir, seqScanPattern(template, loc, alphaValuePattern), func(s string) int {
		return alphaToInt(strings.ToLower(s))
	})
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
	label := ""
	if alloc != nil {
		label = intToAlpha(alloc.nextAlpha(destDir, template))
	} else {
		label = ResolveSeqAlpha(destDir, template)
	}
	return seqAlphaToken.ReplaceAllString(template, label)
}

var seqRomanToken = regexp.MustCompile(`\{seq-roman\}`)

// ResolveSeqRoman returns the next roman numeral for destDir, reading the
// numerals already there through the shape of template.
func ResolveSeqRoman(destDir, template string) string {
	return intToRoman(resolveSeqRomanInt(destDir, template))
}

// resolveSeqRomanInt is the 1-based integer behind ResolveSeqRoman, returning 1
// (which maps to "i") when the directory holds no numeral of this shape, or is
// empty or unreadable.
func resolveSeqRomanInt(destDir, template string) int {
	loc := seqRomanToken.FindStringIndex(template)
	if loc == nil {
		return 1
	}
	return scanMaxSeq(destDir, seqScanPattern(template, loc, romanValuePattern), func(s string) int {
		return romanToInt(strings.ToLower(s))
	})
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
	label := ""
	if alloc != nil {
		label = intToRoman(alloc.nextRoman(destDir, template))
	} else {
		label = ResolveSeqRoman(destDir, template)
	}
	return seqRomanToken.ReplaceAllString(template, label)
}
