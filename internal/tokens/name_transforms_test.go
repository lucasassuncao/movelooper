package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testNameTransform defines the structure for test cases of the name transform functions,
// containing the function under test, the input string, and the expected output.
type testNameTransform struct {
	fn   func(string) string
	name string
	in   string
	want string
}

// testNameTransformTestCases defines a set of test cases for all name transform functions,
// covering basic usage, special characters, edge cases, and empty inputs.
var testNameTransformTestCases = []testNameTransform{
	// nameSlug
	{nameSlug, "slug basic", "My File Name", "my-file-name"},
	{nameSlug, "slug specials", "report (final)!", "report-final"},
	{nameSlug, "slug collapse hyphens", "foo--bar", "foo-bar"},
	{nameSlug, "slug trim hyphens", "-hello-", "hello"},
	{nameSlug, "slug empty", "", ""},
	// nameSnake
	{nameSnake, "snake basic", "My File Name", "my_file_name"},
	{nameSnake, "snake specials", "report (final)!", "report_final"},
	{nameSnake, "snake collapse underscores", "foo__bar", "foo_bar"},
	{nameSnake, "snake trim underscores", "_hello_", "hello"},
	{nameSnake, "snake empty", "", ""},
	// nameAlpha
	{nameAlpha, "alpha removes specials", "report 2025 (final)!", "report2025final"},
	{nameAlpha, "alpha keeps alnum", "abc123", "abc123"},
	{nameAlpha, "alpha empty", "", ""},
	{nameAlpha, "alpha only specials", "!@#", ""},
	// nameASCII
	{nameASCII, "ascii accents", "Ação_résumé", "Acao_resume"},
	{nameASCII, "ascii plain", "hello", "hello"},
	{nameASCII, "ascii empty", "", ""},
	{nameASCII, "ascii only non-ascii", "你好", ""},
	// nameInitials
	{nameInitials, "initials spaces", "my vacation photos", "mvp"},
	{nameInitials, "initials mixed sep", "my-vacation_photos", "mvp"},
	{nameInitials, "initials single word", "report", "r"},
	{nameInitials, "initials empty", "", ""},
	// nameReverse
	{nameReverse, "reverse", "photo", "otohp"},
	{nameReverse, "reverse unicode", "café", "éfac"},
	{nameReverse, "reverse empty", "", ""},
}

// TestNameTransforms tests all name transform functions with various inputs to ensure correct output.
func TestNameTransforms(t *testing.T) {
	t.Parallel()
	for _, tt := range testNameTransformTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.fn(tt.in))
		})
	}
}

// testPreProcessNameTrunc defines the structure for test cases of the preProcessNameTrunc function,
// containing the template string, the file name, and the expected output.
type testPreProcessNameTrunc struct {
	template string
	name     string
	want     string
}

// testPreProcessNameTruncTestCases defines a set of test cases for the preProcessNameTrunc function,
// including truncation by rune count (not bytes) and passthrough when no token is present.
var testPreProcessNameTruncTestCases = []testPreProcessNameTrunc{
	{"{name-trunc:4}", "very-long-name", "very"},
	{"{name-trunc:8}", "very-long-name", "very-lon"},
	{"{name-trunc:20}", "short", "short"},
	{"{name-trunc:1}", "abc", "a"},
	{"prefix_{name-trunc:3}.txt", "report", "prefix_rep.txt"},
	{"no-token", "anything", "no-token"},
	// counts runes not bytes
	{"{name-trunc:3}", "café", "caf"},
	{"{name-trunc:2}", "日本語", "日本"},
}

// TestPreProcessNameTrunc tests the preProcessNameTrunc function to ensure it correctly truncates names by rune count.
func TestPreProcessNameTrunc(t *testing.T) {
	t.Parallel()
	for _, tt := range testPreProcessNameTruncTestCases {
		t.Run(tt.template+"/"+tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, preProcessNameTrunc(tt.template, tt.name))
		})
	}
}
