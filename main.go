package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lucasassuncao/movelooper/internal/cmd"
	"github.com/lucasassuncao/movelooper/internal/models"

	"github.com/spf13/viper"
)

func main() {
	v := viper.GetViper()
	if v == nil {
		fmt.Println("viper couldn't be initialized")
		return
	}

	m := &models.Movelooper{
		Viper:          v,
		Logger:         nil,
		Flags:          &models.Flags{},
		CategoryConfig: make([]*models.CategoryConfig, 0),
	}

	root := cmd.RootCmd(m)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		fmt.Printf("failed to run the app. %v\n", err)
		os.Exit(1)
	}
}
