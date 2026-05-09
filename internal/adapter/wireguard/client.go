package wireguard

import (
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Client struct {
	wireguard *wgctrl.Client
	device    *wgtypes.Device
}

func NewClient(infName string) *Client {
	client, _ := wgctrl.New()

	device, _ := client.Device(infName)

	return &Client{
		wireguard: client,
		device:    device,
	}
}

func (c *Client) getLastHandshake() time.Time {
	return wgtypes.Peer{}.LastHandshakeTime
}
