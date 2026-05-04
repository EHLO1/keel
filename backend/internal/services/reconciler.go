package services

// Reconciler moves time foraward.

import (
	"github.com/EHLO1/keel/backend/internal/config"
)

type ReconcilerService struct {
	cfg *config.Config
	// state  *state.State
	// actor  *actor.Actor
	// policy *policy.Policy
	// log    *slog.Logger
}

func NewReconcilerService(cfg *config.Config) *ReconcilerService {
	return &ReconcilerService{
		cfg: cfg,
	}
}

// func (r *ReconcilerService) Run(ctx context.Context) error {
// 	tick := time.NewTicker(r.cfg.TickInterval)
// 	defer tick.Stop()

// 	r.runOnce(ctx)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		case <-tick.C:
// 			r.runOnce(ctx)
// 		}
// 	}
// }

// func (r *ReconcilerService) runOnce(ctx context.Context) {
// 	snap := r.state.Refresh(ctx)
// 	desired := r.policy.Evaluate(snap)
// 	r.log.Info("tick", "snapshot", snap, "desired", desired)
// 	r.actor.Apply(ctx, &desired, snap)
// }
