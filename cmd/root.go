package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// flagCredential holds the value of the --credential / -c persistent flag.
// Commands that need OAuth use this to locate credentials.json.
var flagCredential string

var rootCmd = &cobra.Command{
	Use:   "gcal-sync",
	Short: "Sync Google Calendar events from one calendar to another",
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&flagCredential, "credential", "c", "",
		"path to Google OAuth credentials.json (required when installed via `go install`)",
	)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(syncCmd)
}
