package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/EHLO1/keel/backend/internal/bootstrap"
	"github.com/EHLO1/keel/backend/internal/signals"
)

var rootCmd = &cobra.Command{
	Use:          "keel",
	Long:         "Keel - 2-Node Backend Orchestrator.",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		err := bootstrap.Bootstrap(cmd.Context())
		if err != nil {
			slog.Error("Failed to run Keel", "error", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	ctx := signals.SignalContext(context.Background())

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}
