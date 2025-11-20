package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucasassuncao/movelooper/internal/core"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/spf13/cobra"
)

// WatchCmd defines the "watch" command to monitor directories and move files in real-time
func WatchCmd(m *models.Movelooper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor folders and move files in real-time",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			if err := preRunHandler(m, configPath); err != nil {
				return err
			}

			// Create a context that cancels on interrupt signals
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				m.Logger.Info("Received interrupt signal, shutting down watcher...")
				cancel()
			}()

			return core.StartWatcher(ctx, m)
		},
	}

	cmd.Flags().StringP("config", "c", "", "Path to configuration file")
	return cmd
}
