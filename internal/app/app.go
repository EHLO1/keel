package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/EHLO1/keel/internal/actor"
	"github.com/EHLO1/keel/internal/adapter/docker"
	"github.com/EHLO1/keel/internal/adapter/filesystem"
	"github.com/EHLO1/keel/internal/adapter/httpc"
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
	HTTPClient      *httpc.Client
	DockerClient    *docker.Client
	ICMPClient      *icmp.Client
	SystemdClient   systemd.Client
	NetworkClient   network.Client

	MaintenanceMode *filesystem.MaintenanceFlag
	StandbySignal   *filesystem.StandbySignal
	VRRPRole        *filesystem.VRRPRole
	StateFile       *filesystem.StateFile

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

	// Initialize Adapters / Clients
	nw, err := network.NewClient(cfg.VRRPVirtualIP, cfg.WireguardInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network client %w", err)
	}

	// Lookup local WireGuard IP
	wrgLocalIP, err := nw.ObserveWireguardIP()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve global IPv4 for interface %s: %w", cfg.WireguardInterface, err)
	}

	// Initialize WireGuard and dynamically identify peers
	wrg := wireguard.NewClient(cfg.WireguardInterface, wrgLocalIP, logger.With("component", "wireguard"))

	pg := postgres.NewClient(ctx, cfg.PostgresAddress(), logger.With("component", "postgres"))
	vk := valkey.NewClient(ctx, cfg.ValkeyAddress(), cfg.ValkeyPassword, cfg.ValkeyDB, logger.With("component", "valkey"))
	hc := httpc.NewClient(cfg.APIPort, logger.With("component", "httpc"))

	ic, err := icmp.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize icmp client: %w", err)
	}

	dr, err := docker.NewClient(ctx, cfg.PostgresVolumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	sys, err := systemd.NewClient(ctx, logger.With("component", "systemd"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize systemd client %w", err)
	}

	// Filesystem Client - Initialize Paths and Files
	mm := filesystem.NewMaintenanceFlag(cfg.MaintenanceFlagPath, cfg.MaintenanceFlagFile, logger.With("component", "fsMaintenanceFlag"))
	ss := filesystem.NewStandbySignal(cfg.StandbySignalFile)
	vr := filesystem.NewVRRPRole(cfg.VRRPRolePath, cfg.VRRPRoleFile)
	sf := filesystem.NewStateFile(cfg.StateFilePath, cfg.StateFile, logger.With("component", "fsStateFile"))

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

	// Initialize State Service (Snapshots)
	systemdServiceList := []string{"docker.service", "keepalived.service", "wireguard.service"}
	st, err := state.NewService(state.Dependencies{
		PG:      pg,
		VK:      vk,
		WRG:     wrg,
		HC:      hc,
		DR:      dr,
		IC:      ic,
		NW:      nw,
		Sys:     sys,
		MM:      mm,
		SS:      ss,
		VR:      vr,
		SF:      sf,
		SvcList: systemdServiceList,
		Log:     logger.With("component", "stateService"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state service: %w", err)
	}

	// Initialize Policy & Actor Services
	pol, err := policy.NewEvaluator(systemdServiceList, logger.With("component", "policyEvaluator"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy evaluator service: %w", err)
	}
	act, err := actor.NewEnforcer(actor.Dependencies{
		Config: cfg,
		ST:     st,
		PG:     pg,
		VK:     vk,
		DR:     dr,
		Sys:    sys,
		SS:     ss,
		SF:     sf,
		Log:    logger.With("component", "actor"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize actor enforcer service: %w", err)
	}

	// Initialize Reconciler (Timing & Controller)
	rec, err := reconciler.NewService(st, pol, act, nw, logger.With("component", "reconciler"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize reconciler service: %w", err)
	}

	stapi := api.NewServer(cfg.APIPort, st)

	return &App{
		Config:            cfg,
		PostgresClient:    pg,
		ValkeyClient:      vk,
		WireguardClient:   wrg,
		HTTPClient:        hc,
		DockerClient:      dr,
		ICMPClient:        ic,
		SystemdClient:     sys,
		NetworkClient:     nw,
		MaintenanceMode:   mm,
		StandbySignal:     ss,
		VRRPRole:          vr,
		StateFile:         sf,
		PolicyEvaluator:   pol,
		ActorEnforcer:     act,
		StateService:      st,
		ReconcilerService: rec,
		APIServer:         stapi,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return a.ReconcilerService.Start(gCtx)
	})

	g.Go(func() error {
		return a.APIServer.Start(gCtx)
	})

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
