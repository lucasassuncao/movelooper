package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testValidateTemplate defines a structure for test cases of the ValidateTemplate function,
// containing the name of the test case, the template string to validate, and whether an error is expected.
type testValidateTemplate struct {
	name     string
	template string
	wantErr  bool
}

// testValidateTemplateTestCases defines a set of test cases for the ValidateTemplate function,
// covering various template strings and their expected validity.
var testValidateTemplateTestCases = []testValidateTemplate{
	{
		"all known static tokens",
		"{name}{ext}{ext-upper}{ext-lower}{ext-reverse}{name-slug}{name-snake}{name-upper}" +
			"{name-lower}{name-alpha}{name-ascii}{name-initials}{name-reverse}{mod-year}{mod-month}" +
			"{mod-day}{mod-date}{mod-weekday}{created-year}{created-month}{created-day}{created-date}" +
			"{year}{month}{day}{date}{weekday}{hour}{minute}{second}{timestamp}{size-range}{category}" +
			"{hostname}{username}{os}{seq-alpha}{seq-roman}{md5}",
		false,
	},
	{"empty template", "", false},
	{"valid single token", "{ext}", false},
	{"valid composite", "{mod-date}_{name}.{ext}", false},
	{"unknown token", "{unknown}", true},
	{"mixed valid and unknown", "{name}_{foo}", true},
	{"partial brace no token", "hello world", false},
	// seq
	{"seq no padding", "{seq}", false},
	{"seq with padding 4", "{seq:4}", false},
	{"seq with padding 1", "{seq:1}", false},
	{"seq with padding 20", "{seq:20}", false},
	{"seq zero padding invalid", "{seq:0}", true},
	{"seq padding too large", "{seq:21}", true},
	{"seq non-numeric padding", "{seq:abc}", true},
	{"seq empty padding", "{seq:}", true},
	{"seq combined with other tokens", "{seq:4}_{name}.{ext}", false},
	// name-trunc
	{"name-trunc valid", "{name-trunc:10}", false},
	{"name-trunc min", "{name-trunc:1}", false},
	{"name-trunc max", "{name-trunc:255}", false},
	{"name-trunc zero", "{name-trunc:0}", true},
	{"name-trunc over max", "{name-trunc:256}", true},
	{"name-trunc non-numeric", "{name-trunc:abc}", true},
	{"name-trunc empty param", "{name-trunc:}", true},
	// md5
	{"md5 default", "{md5}", false},
	{"md5:N valid", "{md5:8}", false},
	{"md5:N min", "{md5:1}", false},
	{"md5:N max", "{md5:32}", false},
	{"md5:N zero", "{md5:0}", true},
	{"md5:N over max", "{md5:33}", true},
	{"md5:N non-numeric", "{md5:abc}", true},
	{"md5:N empty param", "{md5:}", true},
	// sha256
	{"sha256:N valid", "{sha256:16}", false},
	{"sha256:N min", "{sha256:1}", false},
	{"sha256:N max", "{sha256:64}", false},
	{"sha256:N zero", "{sha256:0}", true},
	{"sha256:N over max", "{sha256:65}", true},
	{"sha256:N non-numeric", "{sha256:abc}", true},
	{"sha256:N empty param", "{sha256:}", true},
	{"sha256 bare no param", "{sha256}", true},
	{"stops at first invalid", "{name}_{foo}_{bar}", true},
}

// TestValidateTemplate tests the ValidateTemplate function with various template strings
// to ensure it correctly identifies valid and invalid templates.
func TestValidateTemplate(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidateTemplateTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testRenameOnlyToken defines a structure for test cases of the RenameOnlyToken function,
// containing the name, the template, and the first rename-only token expected (or "").
type testRenameOnlyToken struct {
	name     string
	template string
	want     string
}

// testRenameOnlyTokenTestCases covers tokens that are valid in rename but must be
// rejected in organize-by (sequence and hash families), plus tokens that are fine in both.
var testRenameOnlyTokenTestCases = []testRenameOnlyToken{
	{"organize-by-safe tokens", "{ext}/{mod-year}/{name}", ""},
	{"name-trunc is allowed", "{name-trunc:10}", ""},
	{"username is allowed", "{username}/{year}", ""},
	{"seq", "{seq}/{ext}", "{seq}"},
	{"seq padded", "{seq:4}", "{seq:4}"},
	{"seq-alpha", "{seq-alpha}", "{seq-alpha}"},
	{"seq-roman", "{seq-roman}", "{seq-roman}"},
	{"md5", "{md5}", "{md5}"},
	{"md5 param", "{md5:8}", "{md5:8}"},
	{"sha256 param", "{sha256:16}", "{sha256:16}"},
	{"returns first match", "{ext}/{md5}/{seq}", "{md5}"},
}

// TestRenameOnlyToken tests that RenameOnlyToken flags sequence/hash tokens that
// ResolveGroupBy cannot resolve, while leaving organize-by-safe tokens untouched.
func TestRenameOnlyToken(t *testing.T) {
	t.Parallel()
	for _, tt := range testRenameOnlyTokenTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, RenameOnlyToken(tt.template))
		})
	}
}
