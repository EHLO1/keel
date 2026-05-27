package valkey

import "time"

type Role string

const (
	RoleUnknown Role = ""
	RolePrimary Role = "primary"
	RoleReplica Role = "replica"
)

type InfoMap map[string]string

type ValkeyState struct {
	ObservedAt time.Time `json:"observed_at"`
	Reachable  bool      `json:"reachable"`

	// Role is Primary or Replica
	Role                 string        `json:"role"`
	ConnectedReplicas    int           `json:"connected_replicas"`
	ReplicasWaitingPsync int           `json:"replicas_waiting_psync"`
	PrimaryFailoverState FailoverState `json:"primary_failover_state"`
	PrimaryReplId        string        `json:"primary_repl_id"`
	SecondReplId         string        `json:"second_repl_id"`
	PrimaryReplOffset    int64         `json:"primary_repl_offset"`
	SecondReplOffset     int64         `json:"second_repl_offset"`
	ReplBacklogActive    bool          `json:"repl_backlog_active"`
	ReplBacklogSize      int64         `json:"repl_backlog_size"`

	// Role is Primary
	Replicas []Replica `json:"replicas,omitempty"`

	// Role is Replica
	PrimaryLinkStatus       string `json:"primary_link_status"`
	PrimaryLastIOSecondsAgo int    `json:"primary_last_io_seconds_ago"`
	PrimarySyncInProgress   bool   `json:"primary_sync_in_progress"`
	ReplicaReadReplOffset   int64  `json:"replica_read_repl_offset"`
	ReplicaReplOffset       int64  `json:"replica_repl_offset"`
	ReplicaReplBufferSize   int64  `json:"replica_repl_buffer_size"`
	ReplicaPriority         int    `json:"replica_priority"`
	ReplicaReadOnly         bool   `json:"replica_read_only"`
}

type Replica struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	State  string `json:"state"`
	Offset int64  `json:"offset"`
	Lag    int    `json:"lag"`
}

type FailoverState string

const (
	None           FailoverState = "no-failover"
	WaitingForSync FailoverState = "waiting-for-sync"
	InProgress     FailoverState = "failover-in-progress"
)
