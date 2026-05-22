package state

import (
	"time"

	"github.com/EHLO1/keel/internal/adapter/httpc"
	"github.com/EHLO1/keel/internal/adapter/postgres"
	"github.com/EHLO1/keel/internal/adapter/systemd"
	"github.com/EHLO1/keel/internal/adapter/valkey"
	"github.com/EHLO1/keel/internal/adapter/wireguard"
)

type NodeRole string
type UpDown string
type Health string

const (
	NodePrimary   NodeRole = "primary"
	NodeSecondary NodeRole = "secondary"

	Up   UpDown = "up"
	Down UpDown = "down"

	Healthy   Health = "healthy"
	Unhealthy Health = "unhealthy"
)

type Snapshot struct {
	// Measured or Requested
	CapturedAt        time.Time                `json:"captured_at"`
	Hostname          string                   `json:"hostname"`
	VRRPRole          string                   `json:"vrrp_role"`           // filesystem
	OwnsVIP           bool                     `json:"owns_vip"`            // network
	Postgres          postgres.PostgresState   `json:"postgres"`            // postgres
	Valkey            valkey.ValkeyState       `json:"valkey"`              // valkey
	WireguardState    wireguard.WireguardState `json:"wireguard_state"`     // wireguard
	PostgresInStandby bool                     `json:"postgres_in_standby"` // filesystem
	MaintenanceMode   bool                     `json:"maintenance_mode"`    // filesystem
	Systemd           systemd.ServiceStatus    `json:"systemd"`             // systemd
	PeerKeelInstances []PeerKeelInstance       `json:"peer_keel_instances"`

	// Derived or Aggregated
	PeerDownStrikes         int      `json:"peer_down_strikes"`
	NodeRole                NodeRole `json:"node_role"`
	WireGuardTunnelState    Health   `json:"wireguard_tunnel_state"`
	LoadBalancerIsReachable bool     `json:"load_balancer_is_reachable"` // icmp
}

type PeerKeelInstance struct {
	WireguardIP           string          `json:"wireguard_ip"`
	RealIP                string          `json:"real_ip"`
	PingableOverWireguard bool            `json:"pingable_over_wireguard"`
	PingableOverReal      bool            `json:"pingable_over_real"`
	APISnapshot           httpc.PeerState `json:"api_snapshot"`
	APISnapshotAgeSeconds float64         `json:"api_snapshot_age_seconds"`
	APIReachable          bool            `json:"api_reachable"`
}
