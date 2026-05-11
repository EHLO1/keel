package actor

import (
	"context"

	"github.com/EHLO1/keel/internal/types"
)

type ActorEnforcer struct {
}

func NewEnforcer() (*ActorEnforcer, error) {
	return &ActorEnforcer{}, nil
}

func (e *ActorEnforcer) Apply(ctx context.Context, desiredState *types.DesiredState) error {
	return nil
}
