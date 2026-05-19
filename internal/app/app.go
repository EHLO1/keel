package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EHLO1/keel/internal/actor"
	"github.com/EHLO1/keel/internal/adapter/docker"
	"github.com/EHLO1/keel/internal/adapter/filesystem"
	"github.com/EHLO1/keel/internal/adapter/http"
	"github.com/EHLO1/keel/internal/adapter/icmp"
	"github.com/EHLO1/keel/internal/adapter/network"
	"github.com/EHLO1/keel/internal/adapter/postgres"
	"github.com/EHLO1/keel/internal/adapter/systemd"
	"github.com/EHLO1/keel/internal/adapter/valkey"
	"github.com/EHLO1/keel/internal/adapter/wireguard"
	"github.com/EHLO1/keel/internal/api"
	"github.com/EHLO1/keel/internal/config"
	"github.com/EHLO1/keel/internal/policy"
	"github.com/EHLO1/keel/internal/reconciler"
	"github.com/EHLO1/keel/internal/state"
	"golang.org/x/sync/errgroup"
)

type App struct {
	Config *config.Config

	// Clients
	PostgresClient  *postgres.Client
	ValkeyClient    *valkey.Client
	WireguardClient *wireguard.Client
	HTTPClient      *http.Client
	DockerClient    *docker.Client
	ICMPClient      *icmp.Client
	SystemdClient   systemd.Client
	NetworkClient   network.Client

	MaintenanceMode *filesystem.MaintenanceFlag
	StandbySignal   *filesystem.StandbySignal
	VRRPRole        *filesystem.VRRPRole

	// Core Logic
	PolicyEvaluator *policy.Evaluator
	ActorEnforcer   actor.Enforcer

	// Long-Running Workers
	StateService      *state.Service
	ReconcilerService reconciler.Service
	APIServer         *api.Server
}

func Initialize(ctx context.Context, cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── WireGuard ────────────────────────────────────────────────────────────
	// ─────────────────────────────────────────────────────────────────────────
	// ── Initialize Adapters / Clients ────────────────────────────────────────
	pg := postgres.NewClient(ctx, cfg.PostgresAddress(), logger.With("component", "postgres"))
	vk := valkey.NewClient(ctx, cfg.ValkeyAddress(), cfg.ValkeyPassword, cfg.ValkeyDB, logger.With("component", "valkey"))
	wg := wireguard.NewClient(cfg.WireguardInterface, logger.With("component", "wireguard"))
	http := http.NewClient()

	docker, err := docker.NewClient(ctx, cfg.PostgresVolumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	icmp, err := icmp.NewClient(cfg.PingTargetList())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize icmp client: %w", err)
	}

	network, err := network.NewClient(cfg.VRRPVirtualIP, cfg.WireguardInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network client %w", err)
	}

	sys, err := systemd.NewClient(ctx, logger.With("component", "systemd"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize systemd client %w", err)
	}

	// Filesystem Client - Initialize Paths and Files
	mm := filesystem.NewMaintenanceFlag(cfg.MaintenanceFlagPath, cfg.MaintenanceFlagFile)
	ss := filesystem.NewStandbySignal(cfg.StandbySignalFile)
	vr := filesystem.NewVRRPRole(cfg.VRRPRolePath, cfg.VRRPRoleFile)

	// Preflight Checks
	// preflight := preflightChecks(appCtx, cfg, appServices)
	// for _, w := range preflight.Warnings {
	// 	slog.WarnContext(appCtx, "preflight warning", "issue", w)
	// }
	// if len(preflight.Errors) > 0 {
	// 	for _, e := range preflight.Errors {
	// 		slog.ErrorContext(appCtx, "preflight error", "issue", e)
	// 	}
	// 	return fmt.Errorf("preflight failed: %d errors", len(preflight.Errors))
	// }

	// if err := runServices(ctx, cancel, appServices); err != nil {
	// 	return err
	// }

	// slog.InfoContext(appCtx, "Keel shutdown complete.")

	// return nil

	// ── Initialize Core Logic ────────────────────────────────────────────────
	policy, err := policy.NewEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy evaluator service: %w", err)
	}
	actor, err := actor.NewEnforcer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize actor enforcer service: %w", err)
	}

	// ── Initialize Long-Running Workers ──────────────────────────────────────
	state, err := state.NewService(
		pg,
		vk,
		wg,
		http,
		docker,
		icmp,
		network,
		sys,
		mm,
		ss,
		vr,
		logger.With("component", "stateService"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state service: %w", err)
	}

	reconciler, err := reconciler.NewService(state, policy, actor, net)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize reconciler service: %w", err)
	}

	api := api.NewServer(cfg.APIPort)

	return &App{
		Config:            cfg,
		PostgresClient:    pg,
		ValkeyClient:      vk,
		WireguardClient:   wg,
		HTTPClient:        http,
		DockerClient:      docker,
		ICMPClient:        icmp,
		SystemdClient:     sys,
		NetworkClient:     network,
		MaintenanceMode:   mm,
		StandbySignal:     ss,
		VRRPRole:          vr,
		PolicyEvaluator:   policy,
		ActorEnforcer:     actor,
		StateService:      state,
		ReconcilerService: reconciler,
		APIServer:         api,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	// errgroup tied to the signal context
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return a.StateService.Start(gCtx)
	})

	g.Go(func() error {
		return a.ReconcilerService.Start(gCtx)
	})

	g.Go(func() error {
		return a.APIServer.Start(gCtx)
	})

	// If any of the above return an error, gCtx is canceled,
	// triggering a graceful shutdown for everything else.
	return g.Wait()
}

// func preflightChecks(appCtx context.Context, cfg *config.Config, services *Services) PreflightCheckResult {
// 	var pre PreflightCheckResult

// 	// Hard: wg0 must exist.
// 	if err := services.WireGuard.CheckWireguardInterface(appCtx); err != nil {
// 		pre.Errors = append(pre.Errors, fmt.Sprintf("state file directory not writable: %v", err))
// 	}

// 	// Hard: state file directory must be writable.
// 	if err := services.Filesystem.CheckWritableDir(filepath.Dir(cfg.StateFile)); err != nil {
// 		pre.Errors = append(pre.Errors, fmt.Sprintf("state file directory not writable: %v", err))
// 	}

// 	// Hard: local PG must be reachable (we'll need it from tick 1).
// 	if _, err := services.Postgres.CheckLocalRole(appCtx); err != nil {
// 		pre.Errors = append(pre.Errors, fmt.Sprintf("local postgres unreachable: %v", err))
// 	}

// 	// Hard: local Valkey must be reachable.
// 	if _, _, err := services.Valkey.CheckLocalRole(appCtx); err != nil {
// 		pre.Errors = append(pre.Errors, fmt.Sprintf("local valkey unreachable: %v", err))
// 	}

// 	// Soft: peer reachability is informational at startup.
// 	if r := services.HTTP.CheckPeerConnectivity(appCtx, cfg.PeerQueueHealthPath); !r.OK {
// 		msg := fmt.Sprintf("peer queue-health unreachable: status=%d latency=%s",
// 			r.Status, r.Latency)
// 		if r.Err != nil {
// 			msg += " err=" + r.Err.Error()
// 		}
// 		pre.Warnings = append(pre.Warnings, msg)
// 	}

// 	return pre
// }
