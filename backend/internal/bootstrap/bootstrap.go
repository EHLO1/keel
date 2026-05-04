package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/EHLO1/keel/backend/internal/config"
)

func Bootstrap(ctx context.Context) error {
	_ = godotenv.Load()
	cfg := config.Load()

	slog.InfoContext(ctx, "Keel is starting...")

	appCtx, cancelApp := context.WithCancel(ctx)
	defer cancelApp()

	err := startServer(appCtx, server)
	if err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	return nil
}

func startServer(appCtx context.Context, server *http.Server) error {
	go func() {
		var err error

		err = server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(appCtx, "Failed to start server", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.InfoContext(appCtx, "Received shutdown signal")
	case <-appCtx.Done():
		slog.InfoContext(appCtx, "Context canceled")
	}

	// Use background context for shutdown as appCtx is already canceled
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:contextcheck
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck
		slog.ErrorContext(shutdownCtx, "Server forced to shutdown", "error", err) //nolint:contextcheck
		return err
	}

	slog.InfoContext(shutdownCtx, "Server stopped gracefully") //nolint:contextcheck

	return nil
}

func newConfiguredHTTPClient(cfg *config.Config) *http.Client {
	if cfg.HTTPClientTimeout > 0 {
		return httputils.NewHTTPClientWithTimeout(time.Duration(cfg.HTTPClientTimeout) * time.Second)
	}
	return httputils.NewHTTPClient()
}
