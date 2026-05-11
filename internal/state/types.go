package state

import (
	"context"

	"github.com/EHLO1/keel/internal/types"
)

type Service interface {
	Start(ctx context.Context) error
	Refresh(ctx context.Context) *types.Snapshot
	Current() *types.Snapshot
}
