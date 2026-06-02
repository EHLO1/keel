package policy

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/EHLO1/keel/internal/state"
)

type Evaluator struct {
	role           atomic.Int32
	debouncer      map[Qualifier]*debouncer
	published      atomic.Pointer[state.Snapshot] // Same snapshot, just with new and improved Role
	pgLagThreshold int64
	log            *slog.Logger
}

func NewEvaluator(log *slog.Logger, pgLagThreshold int64) (*Evaluator, error) {
	return &Evaluator{
		pgLagThreshold: pgLagThreshold,
		log:            log,
	}, nil
}

type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DesiredState struct {
	Postgres        string    `json:"postgres"`
	Valkey          string    `json:"valkey"`
	ValkeyReplicaOf *HostPort `json:"valkey_replica_of,omitempty"`
	StateFile       string    `json:"state_file,omitempty"`
	Rationale       string    `json:"rationale"`
}

func (e *Evaluator) Role() Role {
	return Role(e.role.Load())
}

func (e *Evaluator) setRole(r Role) {
	e.role.Store(int32(r))
}

func (e *Evaluator) Evaluate(snap *state.Snapshot) *DesiredState {
	now := time.Now()
	current := e.role.Load()

	bfq := e.baseFitnessQualifier(snap, Role(current), now)
	r := e.determineRole(bfq, snap, Role(current), now)
	e.role.Store(int32(r))

	pub := *snap
	pub.Role = r.String()
	e.published.Store(&pub)

	// Temp placeholder
	vof := &HostPort{"", 6379}

	return &DesiredState{
		Postgres:        "",
		Valkey:          "",
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

func (e *Evaluator) fitForPrimary(snap *state.Snapshot) Verdict {
	if !snap.OwnsVIP && snap.VRRPRole != "MASTER" {
		return Verdict{false, "not assigned the vrrp vip"}
	}
	if snap.Postgres.Role != "primary" {
		return Verdict{false, "postgres role is not primary"}
	}
	if snap.Valkey.Role != "primary" {
		return Verdict{false, "valkey role is not primary"}
	}
	return Verdict{true, ""}
}

// Standby needs to be all about replication health. A healthy standby must have healthy replication.
// A healthy standby with healthy replication has an active primary. If it doesn't have an active primary,
// then the healthy standby may potentially become the new primary.
func (e *Evaluator) fitForStandby(snap *state.Snapshot) Verdict {
	if snap.Postgres.Role != "replica" {
		return Verdict{false, "postgres role is not replica"}
	}
	if v := pgReplicationHealthy(snap, e.pgLagThreshold); !v.OK {
		return v
	}
	if snap.Valkey.Role != "replica" {
		return Verdict{false, "valkey role is not replica"}
	}
	if v := vkReplicationHealthy(snap); !v.OK {
		return v
	}
	return Verdict{true, ""}
}

func (e *Evaluator) determineRole(bfq Verdict, snap *state.Snapshot, current Role, now time.Time) Role {
	if !bfq.OK {
		return RoleUnhealthy
	}

	switch current {
	case RoleUnhealthy:
		return RoleNotReady

	case RoleNotReady:
		if ffp := e.fitForPrimary(snap); ffp.OK {
			return RolePrimary
		}
		if ffs := e.fitForStandby(snap); ffs.OK {
			return RoleStandby
		}
		return RoleNotReady

	case RolePrimary:
		if ffp := e.fitForPrimary(snap); !ffp.OK {
			return RoleNotReady
		}
		return RolePrimary

	case RoleStandby:
		if ffs := e.fitForStandby(snap); !ffs.OK {
			return RoleNotReady
		}
		return RoleStandby

	default:
		return RoleUnhealthy
	}
}

// Published Snapshots include the derived Role field and are primarily used for the state API.
func (e *Evaluator) PublishedSnapshot() *state.Snapshot {
	return e.published.Load()
}
