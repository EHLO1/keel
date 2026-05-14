package postgres

import "time"

type Role string

const (
	RoleUnknown Role = ""
	RolePrimary Role = "primary"
	RoleReplica Role = "replica"
)

type PostgresState struct {
	Reachable     bool      `json:"reachable"`
	Role          string    `json:"role"`
	InStandbyMode bool      `json:"in_standby_mode,omitempty"`
	CapturedAt    time.Time `json:"captured_at"`

	// Role == Primary
	CurrentWriteLSN string    `json:"current_write_lsn,omitempty"`
	Replicas        []Replica `json:"replicas,omitempty"`

	// Role == Replica
	UpstreamPrimaryLSN string `json:"upstream_primary_lsn,omitempty"` // pg_stat_wal_receiver.latest_end_lsn
	ReceiveLSN         string `json:"receive_lsn,omitempty"`          // pg_last_wal_receive_lsn()
	ReplayLSN          string `json:"replay_lsn,omitempty"`           // pg_last_wal_replay_lsn()
	LagBytes           int64  `json:"lag_bytes,omitempty"`            // pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes
	StreamingActive    bool   `json:"streaming_active,omitempty"`     // pg_stat_wal_receiver row exists
}

type Replica struct {
	ApplicationName string `json:"application_name"`
}
