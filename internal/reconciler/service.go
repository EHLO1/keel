package reconciler

import (
	"context"
	"log/slog"
	"time"

	"github.com/EHLO1/keel/internal/state"
)

type ReconcilerService struct {
	state    *state.Service
	policy   PolicyEvaluator
	actor    ActorEnforcer
	vipEvent VIPWatcher
	log      *slog.Logger
}

func NewService(state *state.Service, policy PolicyEvaluator, actor ActorEnforcer, vipEvent VIPWatcher, log *slog.Logger) (*ReconcilerService, error) {
	return &ReconcilerService{
		state:    state,
		policy:   policy,
		actor:    actor,
		vipEvent: vipEvent,
		log:      log,
	}, nil
}

func (s *ReconcilerService) Start(ctx context.Context) error {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	// Run once immediately before waiting for the first tick
	if err := s.RunOnce(ctx); err != nil {
		// You might want to log this or return it depending on your failure requirements
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			s.RunOnce(ctx)
		}
	}
}

func (s *ReconcilerService) RunOnce(ctx context.Context) error {
	snapshot := s.state.Capture(ctx, s.log)
	desired := s.policy.Evaluate(snapshot)
	s.actor.Apply(ctx, desired)
	return nil
}
