package wireguard

import (
	"log/slog"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Client struct {
	wireguard *wgctrl.Client
	device    *wgtypes.Device
	log       *slog.Logger
}

func NewClient(iface string, log *slog.Logger) *Client {
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
		log:       log,
	}
}

func (c *Client) Observe() PeerHandshakeStatus {
	p := c.device.Peers
	result := PeerHandshakeStatus{
		ObservedAt: time.Now(),
		Peers:      make([]Peer, len(p)),
	}

	for i, peer := range p {
		result.Peers[i] = Peer{
			SourceIP:          net.ParseIP(string(peer.Endpoint.IP)),
			LastHandshakeTime: peer.LastHandshakeTime,
			HandshakeAge:      time.Since(peer.LastHandshakeTime).Seconds(),
		}
	}

	return result
}
