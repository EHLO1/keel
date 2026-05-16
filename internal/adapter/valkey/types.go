package valkey

import "time"

type ValkeyState struct {
	Reachable  bool      `json:"reachable"`
	Role       string    `json:"role"`
	ObservedAt time.Time `json:"observed_at"`

	// Role == Primary
	// CurrentWriteLSN string    `json:"current_write_lsn,omitempty"`
	// Replicas        []Replica `json:"replicas,omitempty"`

}
