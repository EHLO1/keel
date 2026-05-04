package reconciler

// Reconciler moves time foraward.

import (
	"context"
	"log/slog"
	"time"

	"github.com/EHLO1/keel/backend/internal/actor"
	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/policy"
	"github.com/EHLO1/keel/backend/internal/state"
)

type Reconciler struct {
	cfg    *config.Config
	state  *state.State
	actor  *actor.Actor
	policy *policy.Policy
	log    *slog.Logger
}

func (r *Reconciler) Run(ctx context.Context) error {
	tick := time.NewTicker(r.cfg.TickInterval)
	defer tick.Stop()

	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Reconciler) runOnce(ctx context.Context) {
	snap := r.state.Refresh(ctx)
	desired := r.policy.Evaluate(snap)
	r.log.Info("tick", "snapshot", snap, "desired", desired)
	r.actor.Apply(ctx, &desired, snap)
}
