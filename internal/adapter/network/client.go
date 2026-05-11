package network

import (
	"context"
)

// The VIPEvent abstracts OS-specific network events away
type VIPEvent struct {
	IsBound   bool
	Interface string
}

type Client interface {
	WatchVIP(ctx context.Context, ch chan<- VIPEvent) error
}
