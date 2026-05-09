package bootstrap

import (
	"context"

	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Services struct {
	HTTP       *services.HTTPClientService
	Postgres   *services.PostgresService
	Valkey     *services.ValkeyClientService
	WireGuard  *services.WireguardService
	Ping       *services.ICMPService
	Filesystem *services.FilesystemService
	State      *services.StateService
	Policy     *services.PolicyService
	Actor      *services.ActorService
	Reconciler *services.ReconcilerService
}

func initializeServices(ctx context.Context, cfg *config.Config, pgPool *pgxpool.Pool) (svcs *Services, err error) {
	svcs = &Services{}

	svcs.HTTP = services.NewHTTPClientService(cfg)
	svcs.Postgres = services.NewPostgresClientService(cfg, pgPool)
	svcs.Valkey = services.NewValkeyClientService(cfg)
	svcs.WireGuard = services.NewWireguardService(cfg)
	svcs.Ping, err = services.NewICMPService(cfg)
	svcs.Filesystem = services.NewFilesystemService(cfg)
	svcs.Policy = services.NewPolicyService(cfg)
	svcs.Actor = services.NewActorService(cfg, svcs.Policy, svcs.State)
	svcs.State = services.NewStateService(
		cfg,
		svcs.HTTP,
		svcs.Postgres,
		svcs.Valkey,
		svcs.WireGuard,
		svcs.Ping,
		// svcs.Filesystem,
	)
	svcs.Reconciler = services.NewReconcilerService(cfg, svcs.State, svcs.Actor, svcs.Policy)

	return svcs, nil
}
