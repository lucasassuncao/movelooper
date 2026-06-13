package tokens

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testFileSizeRange defines a structure for test cases of the fileSizeRange function,
// containing the file size and the expected category.
type testFileSizeRange struct {
	size int64
	want string
}

// testFileSizeRangeTestCases defines a set of test cases for the fileSizeRange function,
// covering various file sizes and their expected categories.
var testFileSizeRangeTestCases = []testFileSizeRange{
	{0, "tiny"},
	{500 * 1024, "tiny"},
	{50 * 1024 * 1024, "small"},
	{500 * 1024 * 1024, "medium"},
	{2 * 1024 * 1024 * 1024, "large"},
	{sizeThresholdTiny, "small"},
	{sizeThresholdSmall, "medium"},
	{sizeThresholdMedium, "large"},
}

// TestFileSizeRange tests the fileSizeRange function with various file sizes to ensure it categorizes them correctly.
func TestFileSizeRange(t *testing.T) {
	t.Parallel()
	for _, tt := range testFileSizeRangeTestCases {
		t.Run(fmt.Sprintf("%d", tt.size), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fileSizeRange(tt.size))
		})
	}
}
