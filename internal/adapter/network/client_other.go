//go:build !linux

package network

import (
	"context"
	"fmt"

	"github.com/EHLO1/keel/internal/types"
)

type mockClient struct {
	vip string
}

func NewClient(ipAddress string) (Client, error) {
	return &mockClient{vip: ipAddress}, nil
}

func (c *mockClient) WatchVIP(ctx context.Context, eventCh chan<- types.VIPEvent) error {
	fmt.Printf("WARN: VIP Watcher for %s not supported on this OS\n", c.vip)

	// Block until the context is canceled so it acts like a real long-running watcher
	<-ctx.Done()
	return ctx.Err()
}
