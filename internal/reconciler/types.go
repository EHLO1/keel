package reconciler

import (
	"context"

	"github.com/EHLO1/keel/internal/adapter/network"
	"github.com/EHLO1/keel/internal/policy"
	"github.com/EHLO1/keel/internal/state"
)

type StateService interface {
	Refresh(ctx context.Context) *state.Snapshot
	Current() *state.Snapshot
}

type PolicyEvaluator interface {
	Evaluate(snapshot *state.Snapshot) *policy.DesiredState
}

type ActorEnforcer interface {
	Apply(ctx context.Context, desiredState *policy.DesiredState) error
}

type VIPWatcher interface {
	WatchVIP(ctx context.Context, ch chan<- network.VIPEvent) error
}

type Service interface {
	Start(ctx context.Context) error
	RunOnce(ctx context.Context) error
}
