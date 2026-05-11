package reconciler

import (
	"context"

	"github.com/EHLO1/keel/internal/types"
)

type StateService interface {
	Refresh(ctx context.Context) *types.Snapshot
	Current() *types.Snapshot
}

type PolicyEvaluator interface {
	Evaluate(snapshot *types.Snapshot) *types.DesiredState
}

type ActorEnforcer interface {
	Apply(ctx context.Context, desiredState *types.DesiredState) error
}

type VIPWatcher interface {
	WatchVIP(ctx context.Context, ch chan<- types.VIPEvent) error
}

type Service interface {
	Start(ctx context.Context) error
	RunOnce(ctx context.Context) error
}
