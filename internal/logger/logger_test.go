package logger

import "github.com/pterm/pterm"

// The pretty renderer is passed around as a Logger; this breaks the build if
// a pterm upgrade changes a signature the interface relies on.
var _ Logger = (*pterm.Logger)(nil)
