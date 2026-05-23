package httpc

import (
	"time"
)

type PeerState struct {
	CapturedAt              time.Time `json:"captured_at"`
	Hostname                string    `json:"hostname"`
	VRRPRole                string    `json:"vrrp_role"`
	OwnsVIP                 bool      `json:"owns_vip"`
	PostgresInStandby       bool      `json:"postgres_in_standby"`
	MaintenanceMode         bool      `json:"maintenance_mode"`
	LocalState              string    `json:"local_state"`
	LoadBalancerIsReachable bool      `json:"load_balancer_is_reachable"`
}
