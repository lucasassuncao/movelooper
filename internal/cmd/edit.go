package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/spf13/cobra"
)

// EditCmd returns the "edit" command, which opens an interactive TUI editor
// for the movelooper configuration file.
func EditCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:               "edit",
		Short:             "Edit the movelooper configuration file in an interactive TUI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Open the movelooper configuration file in an interactive two-panel TUI editor.

The left panel lists top-level configuration keys; pressing Space opens an
overlay where sub-fields can be toggled, edited, and saved. Ctrl+S writes the
file; Ctrl+Z undoes the last change; q quits.

If --output is not provided, the editor opens the file that the --config flag
points to (or the default path when --config is absent).`,
		Example: `  # Edit the default configuration file
  movelooper edit

  # Edit a specific configuration file
  movelooper edit -o /path/to/movelooper.yaml

  # Edit the file passed via the global --config flag
  movelooper --config /path/to/movelooper.yaml edit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := output
			if configPath == "" {
				// Fall back to the global --config flag.
				configPath, _ = cmd.Root().PersistentFlags().GetString("config")
			}
			if configPath == "" {
				ex, err := os.Executable()
				if err != nil {
					return fmt.Errorf("could not determine executable path: %w", err)
				}
				configPath = filepath.Join(filepath.Dir(ex), "conf", "movelooper.yaml")
			}

			return editor.Run(editor.Config{
				Path:    configPath,
				Schema:  &models.Config{},
				Title:   "movelooper",
				Presets: configPresetSource{},
			})
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Path to the configuration file to edit (default: same as --config or <executable_dir>/conf/movelooper.yaml)")

	return cmd
}
