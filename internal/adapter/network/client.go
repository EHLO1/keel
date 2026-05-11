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
		return nil, fmt.Errorf("invalid IP address: %s", vip)
	}

	return &Client{vip: parsedIP}, nil
}

type VIPEvent struct {
	IsBound   bool
	Interface string
}

func (c *Client) WatchVIP(ctx context.Context, eventCh chan<- VIPEvent) error {
	// updateCh receives raw kernel events
	updateCh := make(chan netlink.AddrUpdate, 64)
	// doneCh tells netlink to stop sending events and clean up the socket
	doneCh := make(chan struct{})

	// Subscribe to VIP Changes
	if err := netlink.AddrSubscribe(updateCh, doneCh); err != nil {
		return fmt.Errorf("failed to subscribe to netlink: %w", err)
	}

	defer close(doneCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updateCh:
			if update.LinkAddress.IP == nil {
				continue
			}
			if !update.LinkAddress.IP.Equal(c.vip) {
				continue
			}

			name := ""
			if link, err := netlink.LinkByIndex(update.LinkIndex); err == nil {
				name = link.Attrs().Name
			}

			select {
			case eventCh <- VIPEvent{IsBound: update.NewAddr, Interface: name}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
