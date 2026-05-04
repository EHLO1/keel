package services

import (
	"context"

	"github.com/EHLO1/keel/backend/internal/config"
)

type ActorService struct {
	cfg    *config.Config
	policy *policy.Policy
	state  *state.State
	// future: feature flags, debug mode, alternate rules
}

func NewActorService(cfg *config.Config, policy *policy.Policy, state *state.State) *ActorService {
	return &ActorService{
		cfg:    cfg,
		policy: policy,
		state:  state,
	}
}

func (a *ActorService) Apply(ctx context.Context, desired *policy.DesiredState, snap *state.Snapshot) error {
	return nil
}
