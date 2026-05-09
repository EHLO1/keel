package icmp

import (
	"golang.org/x/net/icmp"
)

type Client struct {
	icmp *icmp.PacketConn
}

func NewClient() (*Client, error) {
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, err
	}

	return &Client{icmp: conn}, nil
}
