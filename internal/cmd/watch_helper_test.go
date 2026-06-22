package cmd

import (
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestCategoriesWithHooks verifies that only categories defining a before or
// after hook are reported, so watch mode warns about exactly those.
func TestCategoriesWithHooks(t *testing.T) {
	t.Parallel()

	cats := []*models.Category{
		{Name: "no-hooks"},
		{Name: "empty-hooks", Hooks: &models.CategoryHooks{}},
		{Name: "with-before", Hooks: &models.CategoryHooks{Before: &models.CategoryHook{}}},
		{Name: "with-after", Hooks: &models.CategoryHooks{After: &models.CategoryHook{}}},
	}

	assert.Equal(t, []string{"with-before", "with-after"}, categoriesWithHooks(cats))
}
