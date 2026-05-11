package network

import (
	"context"

	"github.com/EHLO1/keel/internal/types"
)

// The VIPEvent abstracts OS-specific network events away

type Client interface {
	WatchVIP(ctx context.Context, ch chan<- types.VIPEvent) error
}
