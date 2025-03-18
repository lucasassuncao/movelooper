package models

// PersistentFlags is a struct that holds the persistent flags that are used by the CLI
type PersistentFlags struct {
	Output     *string
	LogLevel   *string
	ShowCaller *bool
}
