package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucasassuncao/movelooper/internal/cmd"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/updater"
)

// version is set at build time via -ldflags "-X main.version=<tag>"
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// run wires up the application and executes it. It exists so the signal
// handler's cleanup runs on every path: main calls os.Exit on failure, which
// would skip any defer declared alongside it.
func run() error {
	updater.CleanOldBinary()

	m := &models.Movelooper{
		Categories: make([]*models.Category, 0),
	}

	root := cmd.RootCmd(m, version)

	// Ctrl+C (and SIGTERM) cancels the context instead of killing the process
	// outright, so a run in progress can stop moving files and still write the
	// history of what it already moved. Without this, an interrupted run leaves
	// files moved and nothing to undo them with.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return root.ExecuteContext(ctx)
}
