package wireguard

import (
	"time"
)

type WireguardState struct {
	ObservedAt time.Time
	Address    string
	Peers      []Peer
}

type Peer struct {
	Endpoint          string
	AllowedIPs        []string
	LastHandshakeTime time.Time
	HandshakeAge      float64
}
