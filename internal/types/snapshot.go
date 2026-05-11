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
	VRRPRole                 VrrpRole     `json:"vrrp_role"`
	OwnsVIP                  bool         `json:"owns_vip"`
	PostgresRole             PostgresRole `json:"postgres_role"`
	ValkeyRole               ValkeyRole   `json:"valkey_role"`
	WireGuardLastHandshake   time.Time    `json:"wireguard_last_handshake"`
	WireGuardTunnelState     Health       `json:"wireguard_tunnel_state"`
	PeerIsReachableWireGuard bool         `json:"peer_is_reachable_wireguard"`
	PeerIsReachable          bool         `json:"peer_is_reachable"`
	WireGuardHandshakeAge    float64      `json:"wireguard_handshake_age_sec"`
	LoadBalancerIsReachable  bool         `json:"load_balancer_is_reachable"`
	Maintenance              bool         `json:"maintenance"`
	CapturedAt               time.Time    `json:"captured_at"`
	PeerDownStrikes          int          `json:"peer_down_strikes"`
	NodeRole                 NodeRole     `json:"node_role"`
	DockerService            UpDown       `json:"docker_service"`
	WireGuardService         UpDown       `json:"wireguard_service"`
	KeepalivedService        UpDown       `json:"keepalived_service"`
	DockerBackendStatus      Health       `json:"docker_backend_status"`
	DockerFrontendStatus     Health       `json:"docker_frontend_status"`
	PostgresCurrentLSN       string       `json:"postgres_current_lsn,omitempty"`
	PostgresReplayLSN        string       `json:"postgres_replay_lsn,omitempty"`
}
