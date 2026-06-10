// Package cmd contains the command line interface commands for the Movelooper application
package cmd

import "github.com/spf13/cobra"

// initOptions holds the flag values for the init command.
type initOptions struct {
	force       bool
	interactive bool
	list        bool
	template    string
	output      string
	scan        string // path to scan; empty means --scan not provided
}

// InitCmd generates a configuration file
func InitCmd() *cobra.Command {
	opts := initOptions{}

	cmd := &cobra.Command{
		Use:               "init",
		Short:             "Initialize movelooper configuration",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
		Long: `Initialize movelooper configuration file with predefined templates or interactive mode.

Available templates:
  - basic:       Simple configuration with one category (images)
  - images:      Configuration for organizing image files
  - music:       Configuration for organizing music files
  - video:       Configuration for organizing video files
  - books:       Configuration for organizing book/document files
  - archives:    Configuration for organizing archive files
  - installers:  Configuration for organizing installer files
  - regex:       Example using regex name filtering
  - full:        Complete example with multiple categories and all options

Scan mode (--scan):
  Analyzes a directory and generates a config based on the file types found.
  Categories are built from a built-in dictionary (images, videos, audio,
  documents, ebooks, archives, fonts, installers). Only categories with at
  least one matching file are included. An 'everything-else' catch-all category
  is always added at the end, disabled by default.

By default the configuration file is created at: <executable_dir>/conf/movelooper.yaml`,
		Example: `  # Interactive mode (recommended for first time)
  movelooper init -i

  # Use a template
  movelooper init -t media

  # Save to a custom path
  movelooper init -o /path/to/movelooper.yaml

  # Force overwrite existing config
  movelooper init -f

  # Scan a directory and generate a config from detected file types
  movelooper init --scan ~/Downloads
  movelooper init --scan ~/Downloads -o /path/to/movelooper.yaml
  movelooper init --scan ~/Downloads -f`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing configuration file")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Interactive mode with prompts")
	cmd.Flags().BoolVarP(&opts.list, "list", "l", false, "List available templates")
	cmd.Flags().StringVarP(&opts.template, "template", "t", "basic", "Template to use (run --list to see available templates)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Path to write the configuration file (default: <executable_dir>/conf/movelooper.yaml)")
	cmd.Flags().StringVar(&opts.scan, "scan", "", "Scan a directory and generate a config from detected file types")

	return cmd
}
