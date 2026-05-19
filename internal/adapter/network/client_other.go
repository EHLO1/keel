//go:build !linux

package network

import (
	"context"
	"fmt"
)

type mockClient struct {
	vip   string
	iface string
}

func NewClient(ipAddress string, iface string) (Client, error) {
	return &mockClient{
		vip:   ipAddress,
		iface: iface,
	}, nil
}

func (c *mockClient) ObserveVIPOwnership() (bool, error) {
	return false, fmt.Errorf("netlink is not supported on this os")
}

func (c *mockClient) WatchVIP(ctx context.Context, eventCh chan<- VIPEvent) error {
	fmt.Printf("WARN: VIP Watcher for %s not supported on this OS\n", c.vip)

	// Block until the context is canceled so it acts like a real long-running watcher
	<-ctx.Done()
	return ctx.Err()
}
