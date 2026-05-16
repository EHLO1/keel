package actor

import (
	"context"

	"github.com/EHLO1/keel/internal/policy"
)

type Enforcer interface {
	Apply(ctx context.Context, desiredState *policy.DesiredState) error
}
