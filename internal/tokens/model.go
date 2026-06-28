package tokens

import (
	"os"
	"strings"
	"time"
)

// TokenContext carries all inputs needed to resolve any token in a template.
// ResolveGroupBy uses Info, CategoryName, and Now.
// ResolveRename additionally uses DestDir and SourcePath.
type TokenContext struct {
	Info         os.FileInfo
	CategoryName string
	Now          time.Time
	DestDir      string        // required for seq, seq-alpha, seq-roman
	SourcePath   string        // required for {md5}, {sha256:N}
	DryRun       bool          // when true, seq/hash tokens are left as literal placeholders
	SeqAlloc     *SeqAllocator // optional per-batch sequence counter; nil falls back to a directory scan per file
	replacer     *strings.Replacer
}
