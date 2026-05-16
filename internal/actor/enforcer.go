package actor

import (
	"context"

	"github.com/EHLO1/keel/internal/policy"
)

type ActorEnforcer struct {
}

func NewEnforcer() (*ActorEnforcer, error) {
	return &ActorEnforcer{}, nil
}

func (e *ActorEnforcer) Apply(ctx context.Context, desiredState *policy.DesiredState) error {
	return nil
}
