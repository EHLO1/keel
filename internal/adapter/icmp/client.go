package icmp

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Client struct {
	conn    *icmp.PacketConn
	id      int
	seq     int
	mu      sync.Mutex
	targets []net.IP
}

func NewClient(pingTargets []string) (*Client, error) {
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, err
	}

	var icmpTargets = make([]net.IP, len(pingTargets))
	for i, t := range pingTargets {
		icmpTargets[i] = net.ParseIP(t)
	}

	return &Client{
		conn:    conn,
		id:      os.Getpid() & 0xffff,
		targets: icmpTargets,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context, timeout time.Duration, target net.IP) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.seq++
	currentSeq := c.seq

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   c.id,
			Seq:  currentSeq,
			Data: []byte("keel"),
		},
	}

	b, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := c.conn.SetDeadline(deadline); err != nil {
		return err
	}

	// For SOCK_DGRAM ICMP, the destination is wrapped in a UDPAddr.
	if _, err := c.conn.WriteTo(b, &net.UDPAddr{IP: target}); err != nil {
		return err
	}

	reply := make([]byte, 1500)
	for {
		n, _, err := c.conn.ReadFrom(reply)
		if err != nil {
			return err // Triggers on timeout deadline
		}
		parsed, err := icmp.ParseMessage(1 /* ProtocolICMP */, reply[:n])
		if err != nil {
			continue
		}
		if parsed.Type != ipv4.ICMPTypeEchoReply {
			continue
		}
		echo, ok := parsed.Body.(*icmp.Echo)
		if !ok {
			continue
		}
		// Verify the reply matches the request
		if echo.ID == c.id && echo.Seq == currentSeq {
			return nil
		}
		// Stray reply for an earlier sequence — keep reading.
	}
}

func (c *Client) Observe(ctx context.Context, timeout time.Duration) ICMPTargets {
	result := ICMPTargets{
		ObservedAt: time.Now(),
		Targets:    make([]Target, len(c.targets)),
	}

	for i, ip := range c.targets {
		err := c.Ping(ctx, timeout, ip)
		result.Targets[i] = Target{
			IP:        ip,
			Reachable: err == nil,
		}
	}

	return result
}
