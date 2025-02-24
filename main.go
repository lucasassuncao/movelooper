package main

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"movelooper/cmd"
	"movelooper/models"
	"os"
)

func main() {
	v := viper.GetViper()
	if v == nil {
		fmt.Println("viper couldn't be initialized")
		return
	}

	m := &models.Movelooper{
		Viper:       v,
		Logger:      nil,
		Flags:       &models.PersistentFlags{},
		MediaConfig: &models.MediaConfig{},
	}

	root := cmd.RootCmd(m)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		fmt.Printf("failed to run the app. %v\n", err)
		os.Exit(1)
	}
}
