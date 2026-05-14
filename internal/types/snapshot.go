package types

import "time"

type VrrpRole string
type PostgresRole string
type ValkeyRole string
type NodeRole string
type UpDown string
type Health string

const (
	VrrpMaster  VrrpRole = "MASTER"
	VrrpBackup  VrrpRole = "BACKUP"
	VrrpFault   VrrpRole = "FAULT"
	VrrpUnknown VrrpRole = "UNKNOWN"

	PostgresPrimary PostgresRole = "primary"
	PostgresReplica PostgresRole = "replica"
	PostgresUnknown PostgresRole = "unknown"

	ValkeyMaster  ValkeyRole = "master"
	ValkeyReplica ValkeyRole = "replica"
	ValkeyUnknown ValkeyRole = "unknown"

	NodePrimary   NodeRole = "primary"
	NodeSecondary NodeRole = "secondary"

	Up   UpDown = "up"
	Down UpDown = "down"

	Healthy   Health = "healthy"
	Unhealthy Health = "unhealthy"
)

type Snapshot struct {
	CapturedAt               time.Time     `json:"captured_at"`
	PeerDownStrikes          int           `json:"peer_down_strikes"`
	NodeRole                 NodeRole      `json:"node_role"`
	VRRPRole                 VrrpRole      `json:"vrrp_role"`                   // filesystem
	OwnsVIP                  bool          `json:"owns_vip"`                    // network
	Postgres                 PostgresState `json:"postgres"`                    // postgres
	ValkeyRole               ValkeyRole    `json:"valkey_role"`                 // valkey
	WireGuardTunnelState     Health        `json:"wireguard_tunnel_state"`      // wireguard
	WireGuardHandshakeAge    float64       `json:"wireguard_handshake_age_sec"` // wireguard
	PeerIsReachableWireGuard bool          `json:"peer_is_reachable_wireguard"` // icmp
	PeerIsReachable          bool          `json:"peer_is_reachable"`           // icmp
	LoadBalancerIsReachable  bool          `json:"load_balancer_is_reachable"`  // http
	Maintenance              bool          `json:"maintenance"`                 // filesystem
	DockerService            UpDown        `json:"docker_service"`              // systemd
	WireGuardService         UpDown        `json:"wireguard_service"`           // systemd
	KeepalivedService        UpDown        `json:"keepalived_service"`          // systemd
	DockerBackendStatus      Health        `json:"docker_backend_status"`       // docker
	DockerFrontendStatus     Health        `json:"docker_frontend_status"`      // docker
}
