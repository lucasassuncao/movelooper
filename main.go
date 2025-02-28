package main

import (
	"context"
	"fmt"
	"log"
	"movelooper/internal/cmd"
	"movelooper/internal/config"
	"movelooper/internal/models"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func main() {
	v := viper.GetViper()
	if v == nil {
		fmt.Println("viper couldn't be initialized")
		return
	}

	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("error getting executable: %v", err)
		return
	}

	options := []config.ViperOptions{
		config.WithConfigName("movelooper"),
		config.WithConfigType("yaml"),
		config.WithConfigPath("."),
		config.WithConfigPath(filepath.Dir(ex)),
		config.WithConfigPath(filepath.Join(filepath.Dir(ex), "conf")),
	}

	if err = config.InitConfig(v, options...); err != nil {
		log.Fatalf("error initializing configuration: %v", err)
		return
	}

	logger, err := config.ConfigureLogger(v)
	if err != nil {
		fmt.Printf("failed to configure logger: %v\n", err)
		return
	}

	m := &models.Movelooper{
		Viper:       v,
		Logger:      logger,
		Flags:       &models.PersistentFlags{},
		MediaConfig: make([]*models.MediaConfig, 0),
	}

	root := cmd.RootCmd(m)

	err = root.ExecuteContext(context.Background())
	if err != nil {
		fmt.Printf("failed to run the app. %v\n", err)
		os.Exit(1)
	}
}
