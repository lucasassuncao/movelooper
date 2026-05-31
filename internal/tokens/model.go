package tokens

import (
	"os"
	"time"
)

// TokenContext carries all inputs needed to resolve any token in a template.
// ResolveGroupBy uses Info, CategoryName, and Now.
// ResolveRename additionally uses DestDir and SourcePath.
type TokenContext struct {
	Info         os.FileInfo
	CategoryName string
	Now          time.Time
	DestDir      string // required for seq, seq-alpha, seq-roman
	SourcePath   string // required for {md5}, {sha256:N}
}
