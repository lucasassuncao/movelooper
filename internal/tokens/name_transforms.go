package tokens

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	multiHyphen     = regexp.MustCompile(`-{2,}`)
	multiUnderscore = regexp.MustCompile(`_{2,}`)
)

func nameSlug(name string) string {
	s := nameASCII(name)
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = multiHyphen.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func nameSnake(name string) string {
	s := nameASCII(name)
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "_")
	s = multiUnderscore.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

func nameAlpha(name string) string {
	return nonAlphanumeric.ReplaceAllString(name, "")
}

func nameASCII(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, name)
	var b strings.Builder
	for _, r := range result {
		if r <= 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func nameInitials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})
	var b strings.Builder
	for _, w := range words {
		rr := []rune(w)
		if len(rr) > 0 {
			b.WriteRune(unicode.ToLower(rr[0]))
		}
	}
	return b.String()
}

func reverseString(s string) string {
	rr := []rune(s)
	for i, j := 0, len(rr)-1; i < j; i, j = i+1, j-1 {
		rr[i], rr[j] = rr[j], rr[i]
	}
	return string(rr)
}

func nameReverse(name string) string { return reverseString(name) }
