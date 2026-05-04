package bootstrap

import (
	"context"

	"github.com/EHLO1/keel/backend/internal/actor"
	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/policy"
	"github.com/EHLO1/keel/backend/internal/probe"
	"github.com/EHLO1/keel/backend/internal/state"
)

type Reconciler struct {
	cfg    *config.Config
	state  *state.State
	policy *policy.Policy
	actor  *actor.Actor
}

func initializeReconciler(ctx context.Context, cfg *config.Config) (probes *Probes, err error) {
	probes = &Probes{}

	probes.HTTP = probe.NewHTTP(cfg, httpClient)
	probes.Postgres = probe.NewPostgres(cfg)
	probes.Valkey = probe.NewValkey(cfg)
	probes.WireGuard = probe.NewWiregaurd(cfg)
	probes.Ping = probe.NewPinger(cfg)

	return probes, nil
}
