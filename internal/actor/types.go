package actor

import (
	"context"

	"github.com/EHLO1/keel/internal/types"
)

type Enforcer interface {
	Apply(ctx context.Context, desiredState *types.DesiredState) error
}
