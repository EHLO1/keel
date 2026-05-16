package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	envPath string
)

var rootCmd = &cobra.Command{
	Use:          "keel",
	Long:         "Keel - Backend Orchestrator for 2+ Nodes.",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// In a real app, use your logger here
		os.Stderr.WriteString("Failed to run Keel: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func init() {
	// Global flag to specify a custom .env file path
	rootCmd.PersistentFlags().StringVarP(&envPath, "env", "e", ".env", "Path to .env file")
}
