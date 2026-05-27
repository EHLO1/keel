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
	ObservedAt    time.Time `json:"observed_at"`

	// Role == Primary
	CurrentWriteLSN string    `json:"current_write_lsn,omitempty"`
	Replicas        []Replica `json:"replicas,omitempty"` //pg_stat_replication

	// Role == Standby
	UpstreamPrimaryLSN string `json:"upstream_primary_lsn,omitempty"` // pg_stat_wal_receiver.latest_end_lsn
	ReceiveLSN         string `json:"receive_lsn,omitempty"`          // pg_last_wal_receive_lsn()
	ReplayLSN          string `json:"replay_lsn,omitempty"`           // pg_last_wal_replay_lsn()
	LagBytes           int64  `json:"lag_bytes,omitempty"`            // pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes
	LagKnown           bool   `json:"lag_known,omitempty"`
	ReceiverStatus     string `json:"receiver_status,omitempty"`
	StreamingActive    bool   `json:"streaming_active,omitempty"` // pg_stat_wal_receiver row exists
}

// ListReplicas()
type Replica struct {
	ApplicationName string `json:"application_name"`
	State           string `json:"state"`
	SyncState       string `json:"sync_state"`
	SentLSN         string `json:"sent_lsn"`
	WriteLSN        string `json:"write_lsn"`
	FlushLSN        string `json:"flush_lsn"`
	ReplayLSN       string `json:"replace_lsn"`
	LagBytes        int64  `json:"lag_bytes"`
	LagKnown        bool   `json:"lag_known"`
}
