package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/lucasassuncao/movelooper/internal/config"
	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/lucasassuncao/yedit/editor"
	"github.com/lucasassuncao/yedit/theme"
	"github.com/spf13/cobra"
)

// EditCmd returns the "edit" command, which opens an interactive TUI editor
// for the movelooper configuration file.
func EditCmd() *cobra.Command {
	var output string
	var themeName string
	var listThemes bool
	var noSaveConfirm bool
	var noDeleteConfirm bool
	var noValidateOnSave bool

	cmd := &cobra.Command{
		Use:               "edit",
		Short:             "Edit the movelooper configuration file in an interactive TUI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Open the movelooper configuration file in an interactive two-panel TUI editor.

The left panel lists top-level configuration keys; pressing Enter opens the
block editor where sub-fields can be toggled and edited. Ctrl+S writes the
file; Ctrl+U undoes the last change; Ctrl+Y redoes it; Esc quits.

Use --output to write to a different file than the one loaded (e.g. to
produce a new config from an existing template).`,
		Example: `  # Edit the default configuration file
  movelooper edit

  # Edit with the Dracula theme
  movelooper edit --theme dracula

  # List all available themes
  movelooper edit --list-themes

  # Load from --config but save to a new file
  movelooper edit --output /path/to/new.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listThemes {
				names := make([]string, 0, len(theme.All()))
				for name := range theme.All() {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					fmt.Println(name)
				}
				return nil
			}

			all := theme.All()
			theme, ok := all[themeName]
			if !ok {
				return fmt.Errorf("unknown theme %q — run 'movelooper edit --list-themes' to see available themes", themeName)
			}

			loadPath, _ := cmd.Root().PersistentFlags().GetString("config")
			if loadPath == "" {
				ex, err := os.Executable()
				if err != nil {
					return fmt.Errorf("could not determine executable path: %w", err)
				}
				loadPath = filepath.Join(filepath.Dir(ex), "conf", "movelooper.yaml")
			}

			movelooperHints, err := buildMovelooperHints()
			if err != nil {
				return fmt.Errorf("building hint source: %w", err)
			}

			res, err := editor.Run(editor.Config{
				Path:                 loadPath,
				SavePath:             output,
				Schema:               &models.Config{},
				Title:                "movelooper",
				BlockPresets:         MovelooperBlockPresets,
				DocPresets:           MovelooperDocPresets,
				EnableHints:          true,
				Metadata:             movelooperHints,
				Theme:                theme,
				PassthroughKeys:      []string{"import"},
				NoSaveConfirm:        noSaveConfirm,
				NoDeleteConfirm:      noDeleteConfirm,
				NoValidateOnSave:     noValidateOnSave,
				SchemaRecursionDepth: config.MaxFilterNestingDepth - 1,
				Validators:           MovelooperValidators,
			})
			if err != nil {
				return err
			}
			if res.Saved {
				savedTo := loadPath
				if output != "" {
					savedTo = output
				}
				fmt.Println("configuration saved to", savedTo)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Save to this file instead of the loaded config (load path is unchanged)")
	cmd.Flags().StringVar(&themeName, "theme", "dark", "Theme name (run --list-themes to see options)")
	cmd.Flags().BoolVar(&listThemes, "list-themes", false, "List available theme names and exit")
	cmd.Flags().BoolVar(&noSaveConfirm, "no-save-confirm", false, "Skip the 'Save changes?' confirmation dialog")
	cmd.Flags().BoolVar(&noDeleteConfirm, "no-delete-confirm", false, "Skip the 'Remove block?' confirmation dialog")
	cmd.Flags().BoolVar(&noValidateOnSave, "no-validate-on-save", false, "Allow saving even when validators report errors (a warning is shown)")

	return cmd
}
