package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/EHLO1/keel/backend/internal/config"
)

func Bootstrap(ctx context.Context) error {
	_ = godotenv.Load()
	cfg := config.Load()

	slog.InfoContext(ctx, "Keel is starting...")

	// Hook signals immediately so initialization is interruptible.
	appCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize Postgres Client PGX Pool.
	pgPool, err := initilizePGXPool(appCtx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres pgx pool: %w", err)
	}
	defer pgPool.Close()

	appServices, err := initializeServices(appCtx, cfg, pgPool)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	preflight := preflightChecks(appCtx, cfg, appServices)
	for _, w := range preflight.Warnings {
		slog.WarnContext(appCtx, "preflight warning", "issue", w)
	}
	if len(preflight.Errors) > 0 {
		for _, e := range preflight.Errors {
			slog.ErrorContext(appCtx, "preflight error", "issue", e)
		}
		return fmt.Errorf("preflight failed: %d errors", len(preflight.Errors))
	}

	if err := runServices(ctx, cancel, appServices); err != nil {
		return err
	}

	slog.InfoContext(appCtx, "Keel shutdown complete.")

	return nil
}

func initilizePGXPool(appCtx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	// Build the connection string.
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresDB,
	)

	// Build the PGX (github.com/jackc/pgx/v5/pgxpool) Pool config.
	poolConfig, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Create a new PGX Pool using config.
	pool, err := pgxpool.NewWithConfig(appCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Ping the database.
	if err := pool.Ping(appCtx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return pool, nil
}

type PreflightCheckResult struct {
	Errors   []string // hard failures — daemon should refuse to start
	Warnings []string // soft issues — log and continue
}

func preflightChecks(appCtx context.Context, cfg *config.Config, services *Services) PreflightCheckResult {
	var pre PreflightCheckResult

	// Hard: wg0 must exist.
	if err := services.WireGuard.CheckWireguardInterface(appCtx); err != nil {
		pre.Errors = append(pre.Errors, fmt.Sprintf("state file directory not writable: %v", err))
	}

	// Hard: state file directory must be writable.
	if err := services.Filesystem.CheckWritableDir(filepath.Dir(cfg.StateFile)); err != nil {
		pre.Errors = append(pre.Errors, fmt.Sprintf("state file directory not writable: %v", err))
	}

	// Hard: local PG must be reachable (we'll need it from tick 1).
	if _, err := services.Postgres.CheckLocalRole(appCtx); err != nil {
		pre.Errors = append(pre.Errors, fmt.Sprintf("local postgres unreachable: %v", err))
	}

	// Hard: local Valkey must be reachable.
	if _, _, err := services.Valkey.CheckLocalRole(appCtx); err != nil {
		pre.Errors = append(pre.Errors, fmt.Sprintf("local valkey unreachable: %v", err))
	}

	// Soft: peer reachability is informational at startup.
	if r := services.HTTP.CheckPeerConnectivity(appCtx, cfg.PeerQueueHealthPath); !r.OK {
		msg := fmt.Sprintf("peer queue-health unreachable: status=%d latency=%s",
			r.Status, r.Latency)
		if r.Err != nil {
			msg += " err=" + r.Err.Error()
		}
		pre.Warnings = append(pre.Warnings, msg)
	}

	return pre
}

func runServices(appCtx context.Context, cancel context.CancelFunc, services *Services) error {
	reconcilerDone := make(chan error, 1)
	go func() {
		slog.InfoContext(appCtx, "Starting Reconciler...")
		reconcilerDone <- services.Reconciler.Run(appCtx)
		slog.InfoContext(appCtx, "Reconciler stopped.")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.InfoContext(appCtx, "Received shutdown signal", "signal", sig)
	case <-appCtx.Done():
		slog.InfoContext(appCtx, "Context canceled")
	case err := <-reconcilerDone:
		// Reconciler exited on its own — likely a fatal error.
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("reconciler exited unexpectedly: %w", err)
		}
		return nil
	}

	// Cancel the context to signal reconciler to stop.
	cancel()

	// Wait for the reconciler to actually exit, with a timeout so a tick
	// can't hang shutdown forever.
	select {
	case err := <-reconcilerDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("reconciler shutdown error: %w", err)
		}
	case <-time.After(30 * time.Second):
		slog.WarnContext(appCtx, "reconciler did not stop within 30s; forcing exit")
	}
	return nil
}
