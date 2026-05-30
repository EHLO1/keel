package policy

import (
	"log/slog"

	"github.com/EHLO1/keel/internal/state"
)

type Evaluator struct {
	log *slog.Logger
}

func NewEvaluator(log *slog.Logger) (*Evaluator, error) {
	return &Evaluator{
		log: log,
	}, nil
}

func (e *Evaluator) Qualify(snap *state.Snapshot) string {
	var r string

	switch {
	case !meetsBaseQualifiers(snap).OK:
		r = "demote"
	case fitForPrimary(snap).OK:
		r = "promoteToPrimary"
	case fitForStandby(snap).OK:
		r = "promoteToStandby"
	default:
		r = ""
	}

	return r
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
			Valkey:    ValkeyPrimary,
			Rationale: "Active node (owns VIP or VRRP role is MASTER)",
		}
	}

	// Temp placeholder
	vof := &HostPort{"", 6379}

	return &DesiredState{
		Postgres:        PostgresReplica,
		Valkey:          ValkeyReplica,
		ValkeyReplicaOf: vof,
		Rationale:       "Standby node (does not own VIP, VRRP role is " + snapshot.VRRPRole + ")",
	}
}
