package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/EHLO1/keel/internal/app"
	"github.com/EHLO1/keel/internal/config"
)

// cmd/run.go
var runCmd = &cobra.Command{
	Use: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		cfg := config.Load(envPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		keel, err := app.Initialize(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize keel: %w", err)
		}

		return keel.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
