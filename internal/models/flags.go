// Package models defines the data structures and functions related to application configuration.
package models

// Flags is a struct that holds the persistent flags that are used by the CLI
type Flags struct {
	Output     *string
	LogLevel   *string
	ShowCaller *bool
}
