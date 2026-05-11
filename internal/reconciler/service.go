package reconciler

import (
	"context"
	"time"
)

type ReconcilerService struct {
	state    StateService
	policy   PolicyEvaluator
	actor    ActorEnforcer
	vipEvent VIPWatcher
}

func NewService(state StateService, policy PolicyEvaluator, actor ActorEnforcer, vipEvent VIPWatcher) (*ReconcilerService, error) {
	return &ReconcilerService{
		state:    state,
		policy:   policy,
		actor:    actor,
		vipEvent: vipEvent,
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
	snapshot := s.state.Refresh(ctx)
	desired := s.policy.Evaluate(snapshot)
	s.actor.Apply(ctx, desired)
	return nil
}
