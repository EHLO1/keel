package wireguard

import (
	"net"
	"time"
)

type PeerHandshakeStatus struct {
	ObservedAt time.Time
	Peers      []Peer
}

type Peer struct {
	SourceIP          net.IP
	LastHandshakeTime time.Time
	HandshakeAge      float64
}
