package bootstrap

import (
	"context"
	"net/http"

	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/services"
)

type Services struct {
	HTTP       *services.HTTPProbeService
	Postgres   *services.PostgresProbeService
	Valkey     *services.ValkeyProbeService
	WireGuard  *services.WireguardProbeService
	Ping       *services.ICMPProbeService
	State      *services.StateService
	Policy     *services.PolicyService
	Actor      *services.ActorService
	Reconciler *services.ReconcilerService
}

func initializeServices(ctx context.Context, cfg *config.Config, httpClient *http.Client) (svcs *Services, err error) {
	svcs = &Services{}

	svcs.HTTP = services.NewHTTPProbeService(cfg, httpClient)
	svcs.Postgres = services.NewPostgresProbeService(cfg)
	svcs.Valkey = services.NewValkeyProbeService(cfg)
	svcs.WireGuard = services.NewWiregaurdProbeService(cfg)
	svcs.Ping = services.NewICMPProbeService(cfg)
	svcs.Actor = services.NewActorService(cfg, svcs.Policy, svcs.State)
	svcs.State = services.NewStateService(
		cfg,
		svcs.HTTP,
		svcs.Postgres,
		svcs.Valkey,
		svcs.WireGuard,
		svcs.Ping,
	)
	svcs.Reconciler = services.NewReconcilerService(cfg, svcs.State)

	return svcs, nil
}
