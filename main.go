package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lucasassuncao/movelooper/internal/cmd"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/movelooper/internal/updater"

	"github.com/spf13/viper"
)

// version is set at build time via -ldflags "-X main.version=<tag>"
var version = "dev"

func main() {
	updater.CleanOldBinary()

	v := viper.GetViper()
	if v == nil {
		fmt.Println("viper couldn't be initialized")
		return
	}

	m := &models.Movelooper{
		Viper:      v,
		Logger:     nil,
		Flags:      &models.Flags{},
		Categories: make([]*models.Category, 0),
	}

	root := cmd.RootCmd(m, version)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		fmt.Printf("Failed to run the app. %v\n", err)
		os.Exit(1)
	}
}
