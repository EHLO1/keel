package wireguard

import (
	"log/slog"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Client struct {
	wireguard *wgctrl.Client
	device    *wgtypes.Device
	addr      string
	log       *slog.Logger
}

func NewClient(iface string, addr string, log *slog.Logger) *Client {
	client, err := wgctrl.New()
	if err != nil {
		log.Error("could not create wireguard client", "error", err)
		return nil
	}

	device, err := client.Device(iface)
	if err != nil {
		log.Error("could not find interface", "error", err)
		return nil
	}

	return &Client{
		wireguard: client,
		device:    device,
		addr:      addr,
		log:       log,
	}
}

func (c *Client) Observe() WireguardState {
	if c == nil || c.device == nil {
		return WireguardState{
			ObservedAt: time.Now(),
		}
	}
	p := c.device.Peers
	result := WireguardState{
		ObservedAt: time.Now(),
		Address:    c.addr,
		Peers:      make([]Peer, len(p)),
	}

	for i, peer := range p {
		var endpoint string
		if peer.Endpoint != nil {
			endpoint = peer.Endpoint.IP.String()
		}
		allowedIPs := make([]string, 0, len(peer.AllowedIPs))
		for _, ipNet := range peer.AllowedIPs {
			allowedIPs = append(allowedIPs, ipNet.IP.String())
		}
		result.Peers[i] = Peer{
			Endpoint:          endpoint,
			AllowedIPs:        allowedIPs,
			LastHandshakeTime: peer.LastHandshakeTime,
			HandshakeAge:      time.Since(peer.LastHandshakeTime).Seconds(),
		}
	}

	return result
}
