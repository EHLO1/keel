package valkey

import "time"

type Role string

const (
	RoleUnknown Role = ""
	RoleMaster  Role = "master"
	RoleReplica Role = "replica"
)

type InfoMap map[string]string

type ValkeyState struct {
	Reachable        bool      `json:"reachable"`
	Role             string    `json:"role"`
	ObservedAt       time.Time `json:"observed_at"`
	MasterReplOffset int64

	// Role == Primary

	Replicas         []Replica `json:"replicas,omitempty"`
	MasterOffset     int64
	MasterLinkStatus string
}

type Replica struct {
}
