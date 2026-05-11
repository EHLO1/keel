package services

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/EHLO1/keel/backend/internal/config"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ICMPService maintains a long-lived ICMP socket and sends echo requests over wg0.
// Construct once, reuse.
type ICMPService struct {
	cfg    *config.Config
	conn   *icmp.PacketConn
	target *net.IPAddr
	id     int
	seq    int
}

func NewICMPService(cfg *config.Config) (*ICMPService, error) {
	// "udp4" here means SOCK_DGRAM ICMP — the unprivileged variant.
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	target, err := net.ResolveIPAddr("ip4", cfg.WireguardPeerIP)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &ICMPService{
		conn:   conn,
		target: target,
		id:     os.Getpid() & 0xffff,
	}, nil
}

func (p *ICMPService) Close() error { return p.conn.Close() }

// Ping sends one echo request and waits up to timeout for a matching reply.
// Returns nil on success, an error otherwise.
func (p *ICMPService) Ping(ctx context.Context, timeout time.Duration) error {
	p.seq++
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  p.seq,
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
	if err := p.conn.SetDeadline(deadline); err != nil {
		return err
	}

	// For SOCK_DGRAM ICMP, the destination is wrapped in a UDPAddr.
	if _, err := p.conn.WriteTo(b, &net.UDPAddr{IP: p.target.IP}); err != nil {
		return err
	}

	reply := make([]byte, 1500)
	for {
		n, _, err := p.conn.ReadFrom(reply)
		if err != nil {
			return err
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
		if echo.ID == p.id && echo.Seq == p.seq {
			return nil
		}
		// Stray reply for an earlier sequence — keep reading.
	}
}
