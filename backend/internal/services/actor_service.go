package services

import (
	"context"

	"github.com/EHLO1/keel/backend/internal/config"
)

type ActorService struct {
	cfg    *config.Config
	policy *PolicyService
	state  *StateService
	// future: feature flags, debug mode, alternate rules
}

func NewActorService(cfg *config.Config, policy *PolicyService, state *StateService) *ActorService {
	return &ActorService{
		cfg:    cfg,
		policy: policy,
		state:  state,
	}
}

func (a *ActorService) Apply(ctx context.Context, desired *DesiredState, snapshot *Snapshot) error {
	return nil
}
