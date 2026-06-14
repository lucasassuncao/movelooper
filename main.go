package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/lucasassuncao/movelooper/internal/cmd"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/updater"
)

// version is set at build time via -ldflags "-X main.version=<tag>"
var version = "dev"

func main() {
	updater.CleanOldBinary()

	m := &models.Movelooper{
		Categories: make([]*models.Category, 0),
	}

	root := cmd.RootCmd(m, version)

	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}
