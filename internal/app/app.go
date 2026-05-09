package app

import (
	"context"

	"github.com/EHLO1/keel/internal/adapter/postgres"
	"github.com/EHLO1/keel/internal/adapter/valkey"
	"github.com/EHLO1/keel/internal/api"
	"github.com/EHLO1/keel/internal/config"
	"golang.org/x/sync/errgroup"
)

type App struct {
	Config *config.Config

	// Clients
	PostgresClient   *postgres.Client
	ValkeyClient     *valkey.Client
	WireguardClient  *wireguard.Client
	HTTPClient       *http.Client
	DockerClient     *docker.Client
	ICMPClient       *icmp.Client
	FilesystemClient *filesystem.Client
	SystemdClient    *systemd.Client
	NetworkClient    *network.Client

	// Core Logic
	PolicyEvaluator *policy.Evaluator
	ActorEnforcer   *actor.Enforcer

	// Long-Running Workers
	StateService      *state.Service
	ReconcilerService *reconciler.Service
	APIServer         *api.Server
}

func Initialize(ctx context.Context, cfg *config.Config) (*App, error) {

	// Initialize Adapters / Clients
	pg := postgres.NewClient(ctx, cfg.PostgresAddress())
	vk := valkey.NewClient(ctx, cfg.ValkeyAddress(), cfg.ValkeyPassword, cfg.ValkeyDB)
	wg := wireguard.NewClient()
	http := http.NewClient()
	docker := docker.NewClient()
	icmp := icmp.NewClient()
	fs := filesystem.NewClient()
	sys := systemd.NewClient()
	net := network.NewClient()

	// Initialize Core Logic
	policy := policy.NewEvaluator()
	actor := actor.NewEnforcer(pg, vk, wg)

	// Initialize Long-Running Workers
	state := state.NewService()
	reconciler := reconciler.NewService()
	api := api.NewServer(cfg, stateSvc)

	return &App{
		Config:            cfg,
		PostgresClient:    pg,
		ValkeyClient:      vk,
		WireguardClient:   wg,
		HTTPClient:        http,
		DockerClient:      docker,
		ICMPClient:        icmp,
		FilesystemClient:  fs,
		SystemdClient:     sys,
		NetworkClient:     net,
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
