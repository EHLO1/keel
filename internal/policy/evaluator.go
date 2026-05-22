package policy

import (
	"log/slog"

	"github.com/EHLO1/keel/internal/state"
)

type Evaluator struct {
	log *slog.Logger
}

func NewEvaluator(log *slog.Logger) (*Evaluator, error) {
	return &Evaluator{log: log}, nil
}

func (e *Evaluator) Evaluate(snapshot *state.Snapshot) *DesiredState {
	if snapshot.MaintenanceMode {
		return &DesiredState{
			Postgres:  PostgresUnknown,
			Valkey:    ValkeyUnknown,
			Rationale: "Maintenance mode is enabled",
		}
	}

	// Active node: owns VRRP VIP or VRRP state is MASTER
	if snapshot.OwnsVIP || snapshot.VRRPRole == "MASTER" {
		return &DesiredState{
			Postgres:  PostgresPrimary,
			Valkey:    ValkeyMaster,
			Rationale: "Active node (owns VIP or VRRP role is MASTER)",
		}
	}

	// Standby node
	var valkeyReplOf *HostPort
	if e.peerWGIP != "" {
		valkeyReplOf = &HostPort{
			Host: e.peerWGIP,
			Port: e.valkeyPort,
		}
	}

	return &DesiredState{
		Postgres:        PostgresReplica,
		Valkey:          ValkeyReplica,
		ValkeyReplicaOf: valkeyReplOf,
		Rationale:       "Standby node (does not own VIP, VRRP role is " + snapshot.VRRPRole + ")",
	}
}
