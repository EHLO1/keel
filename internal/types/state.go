package types

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
