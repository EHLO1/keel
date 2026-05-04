package actor

import (
	"context"

	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/policy"
	"github.com/EHLO1/keel/backend/internal/state"
)

type Actor struct {
	cfg    *config.Config
	policy *policy.Policy
	state  *state.State
	// future: feature flags, debug mode, alternate rules
}

func (a *Actor) Apply(ctx context.Context, desired *policy.DesiredState, snap *state.Snapshot) error {
	return nil
}
