package network

import (
	"context"
)

// The VIPEvent abstracts OS-specific network events away

type Client interface {
	WatchVIP(ctx context.Context, ch chan<- VIPEvent) error
}

type VIPEvent struct {
	IsBound   bool
	Interface string
}
