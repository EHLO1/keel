package policy

type PostgresRole string
type ValkeyRole string

const (
	PostgresPrimary PostgresRole = "primary"
	PostgresReplica PostgresRole = "replica"
	PostgresUnknown PostgresRole = "unknown"

	ValkeyMaster  ValkeyRole = "master"
	ValkeyReplica ValkeyRole = "replica"
	ValkeyUnknown ValkeyRole = "unknown"
)

type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DesiredState struct {
	Postgres        PostgresRole `json:"postgres"`
	Valkey          ValkeyRole   `json:"valkey"`
	ValkeyReplicaOf *HostPort    `json:"valkey_replica_of,omitempty"`
	StateFile       string       `json:"state_file,omitempty"`
	Rationale       string       `json:"rationale"`
}
