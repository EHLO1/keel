package app

import (
	"context"
	"fmt"

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
	WireguardClient *wireguard.Client //golang.zx2c4.com/wireguard/wgctrl
	HTTPClient      *http.Client
	DockerClient    *docker.Client
	ICMPClient      *icmp.Client
	SystemdClient   *systemd.Client // https://github.com/coreos/go-systemd/v22/dbus
	NetworkClient   network.Client

	MaintenanceMode *filesystem.MaintenanceFlag
	StandbySignal   *filesystem.StandbySignal
	VRRPRole        *filesystem.VRRPRole

	// Core Logic
	PolicyEvaluator policy.Evaluator
	ActorEnforcer   actor.Enforcer

	// Long-Running Workers
	StateService      state.Service
	ReconcilerService reconciler.Service
	APIServer         *api.Server
}

func Initialize(ctx context.Context, cfg *config.Config) (*App, error) {

	// ── WireGuard ────────────────────────────────────────────────────────────
	// ─────────────────────────────────────────────────────────────────────────
	// ── Initialize Adapters / Clients ────────────────────────────────────────
	pg := postgres.NewClient(ctx, cfg.PostgresAddress())
	vk := valkey.NewClient(ctx, cfg.ValkeyAddress(), cfg.ValkeyPassword, cfg.ValkeyDB)
	wg := wireguard.NewClient(cfg.WireguardInterface)
	http := http.NewClient()

	docker, err := docker.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	icmp, err := icmp.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize icmp client: %w", err)
	}

	net, err := network.NewClient(cfg.VRRPVirtualIP)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network client %w", err)
	}

	sys, err := systemd.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize systemd client %w", err)
	}

	// Filesystem Client - Initialize Paths and Files
	mm := filesystem.NewMaintenanceFlag(cfg.MaintenanceFlagPath, cfg.MaintenanceFlagFile)
	ss := filesystem.NewStandbySignal(cfg.StandbySignalFile)
	vr := filesystem.NewVRRPRole(cfg.VRRPRolePath, cfg.VRRPRoleFile)

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
	state, err := state.NewService()
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
		NetworkClient:     net,
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
