package main

import (
	"context"
	"fmt"
	"movelooper/cmd"
	"movelooper/models"
	"os"

	"github.com/spf13/viper"
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
		MediaConfig: nil,
	}

	root := cmd.RootCmd(m)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		fmt.Printf("failed to run the app. %v\n", err)
		os.Exit(1)
	}
}
