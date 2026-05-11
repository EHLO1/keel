//go:build linux

package network

import (
	"context"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type Client struct {
	vip net.IP
}

func NewClient(vip string) (*Client, error) {
	parsedIP := net.ParseIP(vip)
	if parsedIP == nil {
		return nil, fmt.Errorf("IP Address is not valid: %s", vip)
	}

	return &Client{
		vip: parsedIP,
	}, nil
}

// VIPEvent represents a state change for our managed Virtual IP.
type VIPEvent struct {
	IsBound   bool
	Interface string
}

func (c *Client) WatchVIP(ctx context.Context, eventCh chan<- VIPEvent) error {
	// updateCh receives the raw kernel events
	updateCh := make(chan netlink.AddrUpdate)
	// doneCh tells netlink to stop sending events and clean up the socket
	doneCh := make(chan struct{})
	defer close(doneCh)

	// Subscribe to VIP Changes
	if err := netlink.AddrSubscribe(updateCh, doneCh); err != nil {
		return fmt.Errorf("failed to subscribe to netlink: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updateCh:
			if update.LinkAddress.IP.Equal(c.vip) {
				eventCh <- VIPEvent{IsBound: update.NewAddr}
			}
		}
	}
}
