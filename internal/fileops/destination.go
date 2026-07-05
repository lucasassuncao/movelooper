package fileops

import (
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/tokens"
)

// ResolveDestDir resolves the destination directory for one file under the
// category's organize-by template. It is the single source of truth for this
// rule, shared by the real move (MoveFiles), the dry-run preview, and watch
// mode, so the three can never disagree about where a file lands.
func ResolveDestDir(category *models.Category, tctx *tokens.TokenContext) string {
	destDir := category.Destination.Path
	if template := category.Destination.OrganizeBy; template != "" {
		if subdir := tokens.ResolveGroupBy(template, tctx); subdir != "" {
			destDir = filepath.Join(category.Destination.Path, subdir)
		}
	}
	return destDir
}

// ResolveDestination resolves the destination directory (organize-by) and the
// final filename (rename) for one file. It sets tctx.DestDir before resolving
// the rename template, which the seq tokens need to scan for existing numbers.
// It never creates directories or touches the destination; with tctx.DryRun
// set, seq/hash tokens are left as literal placeholders.
func ResolveDestination(category *models.Category, tctx *tokens.TokenContext) (destDir, destName string) {
	destDir = ResolveDestDir(category, tctx)
	tctx.DestDir = destDir
	destName = tokens.ResolveRename(category.Destination.Rename, tctx)
	return destDir, destName
}
