package services

// Reconciler moves time foraward.

import (
	"context"
	"time"

	"github.com/EHLO1/keel/backend/internal/config"
)

type ReconcilerService struct {
	cfg    *config.Config
	state  *StateService
	actor  *ActorService
	policy *PolicyService
}

func NewReconcilerService(cfg *config.Config, state *StateService, actor *ActorService, policy *PolicyService) *ReconcilerService {
	return &ReconcilerService{
		cfg:    cfg,
		state:  state,
		actor:  actor,
		policy: policy,
	}
}

func (r *ReconcilerService) Run(ctx context.Context) error {
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

func (r *ReconcilerService) runOnce(ctx context.Context) {
	snapshot := r.state.Refresh(ctx)
	desired := r.policy.Evaluate(snapshot)
	r.actor.Apply(ctx, &desired, snapshot)
}
