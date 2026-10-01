package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Omitting enabled disables a category, so only an explicit true counts.
func TestCategory_IsEnabled(t *testing.T) {
	t.Parallel()
	yes, no := true, false
	assert.False(t, (&Category{}).IsEnabled(), "absent")
	assert.False(t, (&Category{Enabled: &no}).IsEnabled(), "explicit false")
	assert.True(t, (&Category{Enabled: &yes}).IsEnabled(), "explicit true")
}

func TestArchiveConfig_KeepsSource(t *testing.T) {
	t.Parallel()
	yes, no := true, false
	assert.True(t, (&ArchiveConfig{}).KeepsSource(), "absent keeps the source")
	assert.True(t, (&ArchiveConfig{KeepSource: &yes}).KeepsSource())
	assert.False(t, (&ArchiveConfig{KeepSource: &no}).KeepsSource())
}

func TestCategoryFilter_IsZero(t *testing.T) {
	t.Parallel()
	assert.True(t, CategoryFilter{}.IsZero())
	for name, f := range map[string]CategoryFilter{
		"match": {Match: &MatchFilter{}},
		"age":   {Age: &AgeFilter{}},
		"size":  {Size: &SizeFilter{}},
		"mime":  {Mime: "image/*"},
		"any":   {Any: []CategoryFilter{{}}},
		"all":   {All: []CategoryFilter{{}}},
		"not":   {Not: []CategoryFilter{{}}},
	} {
		assert.False(t, f.IsZero(), name)
	}
}
