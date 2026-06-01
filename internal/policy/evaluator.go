package policy

import (
	"log/slog"
	"time"

	"github.com/EHLO1/keel/internal/state"
)

type Evaluator struct {
	debouncer map[Qualifier]*debouncer
	log       *slog.Logger
}

func NewEvaluator(log *slog.Logger) (*Evaluator, error) {
	return &Evaluator{
		debouncer: map[Qualifier]*deboucer,
		log:       log,
	}, nil
}

type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DesiredState struct {
	Postgres        Role      `json:"postgres"`
	Valkey          Role      `json:"valkey"`
	ValkeyReplicaOf *HostPort `json:"valkey_replica_of,omitempty"`
	StateFile       string    `json:"state_file,omitempty"`
	Rationale       string    `json:"rationale"`
}

func (e *Evaluator) Qualify(snap *state.Snapshot) string {
	var r string

	switch {
	case !baseQualifiers(snap).OK:
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

func (e *Evaluator) Evaluate(snap *state.Snapshot) *DesiredState {
	now := time.Now()

	if bfq := e.baseFitnessQualifier(snap, Role, now); !bfq.OK {
		return nil
	}

	// Temp placeholder
	vof := &HostPort{"", 6379}

	return &DesiredState{
		Postgres:        PostgresReplica,
		Valkey:          ValkeyReplica,
		ValkeyReplicaOf: vof,
		Rationale:       "Standby node (does not own VIP, VRRP role is " + snap.VRRPRole + ")",
	}
}

func (e *Evaluator) baseFitnessQualifier(snap *state.Snapshot, role Role, now time.Time) Verdict {
	if snap.MaintenanceMode {
		return Verdict{false, "maintenance mode active"}
	}

	order := []Qualifier{QSystemd, QPostgres, QValkey, QVRRP, QLoadBalancer}
	for _, q := range order {
		raw := baseFitness(q, snap)

		confirmed := e.debouncer[q].observe(raw, now, gracePeriod(q, role))

		if !confirmed.OK {
			return confirmed
		}
	}
	return Verdict{OK: true}
}
