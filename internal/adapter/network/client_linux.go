//go:build linux

package network

import (
	"context"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type linuxClient struct {
	vip   net.IP
	iface string
}

func NewClient(ipAddress string, iface string) (Client, error) {
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	return &linuxClient{
		vip:   parsedIP,
		iface: iface,
	}, nil
}

func (c *linuxClient) ObserveVIPOwnership() (bool, error) {
	link, err := netlink.LinkByName(c.iface)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to get link %s: %w", c.iface, err)
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return false, fmt.Errorf("failed to list addresses for link %s: %w", c.iface, err)
	}

	for _, addr := range addrs {
		if addr.IP.Equal(c.vip) {
			return true, nil
		}
	}

	return false, nil
}

func (c *linuxClient) WatchVIP(ctx context.Context, eventCh chan<- VIPEvent) error {
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
