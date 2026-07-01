package tokens

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveArchiveName(t *testing.T) {
	now := time.Date(2026, 7, 1, 9, 30, 15, 0, time.UTC)

	assert.Equal(t, "photos", ResolveArchiveName("", "photos", now), "empty template falls back to category")
	assert.Equal(t, "photos_2026-07-01", ResolveArchiveName("{category}_{date}", "photos", now))
	assert.Equal(t, "20260701-093015", ResolveArchiveName("{timestamp}", "x", now))
	// path separators in the resolved name are neutralised to keep a plain filename
	assert.Equal(t, "a_b", ResolveArchiveName("a/b", "x", now))
}
