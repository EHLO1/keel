package policy

import (
	"fmt"

	"github.com/EHLO1/keel/internal/state"
)

type Verdict struct {
	OK     bool
	Reason string
}

type Qualifier int

const (
	QSystemd Qualifier = iota
	QVRRP
	QLoadBalancer
	QPostgres
	QValkey
)

func (q Qualifier) String() string {
	switch q {
	case QSystemd:
		return "systemd"
	case QVRRP:
		return "vrrp"
	case QLoadBalancer:
		return "loadbalancer"
	case QPostgres:
		return "postgres"
	case QValkey:
		return "valkey"
	default:
		return fmt.Sprintf("qualifier(%d)", int(q))
	}
}

func checkPostgres(snap *state.Snapshot) Verdict {
	if !snap.Postgres.Reachable {
		return Verdict{false, "local postgres state is unknown"}
	}
	return Verdict{true, ""}
}

func checkValkey(snap *state.Snapshot) Verdict {
	if !snap.Valkey.Reachable {
		return Verdict{false, "local valkey or redis state is unknown"}
	}
	return Verdict{true, ""}
}

func checkVRRP(snap *state.Snapshot) Verdict {
	if snap.VRRPRole == "FAULT" || snap.VRRPRole == "UNKNOWN" {
		return Verdict{false, "vrrp is not ready"}
	}
	return Verdict{true, ""}
}

func checkSystemd(snap *state.Snapshot) Verdict {
	for _, svc := range snap.Systemd.Services {
		if svc.ActiveState != "active" && svc.SubState != "running" {
			return Verdict{false, fmt.Sprintf("service is not active or is not running: %s", svc)}
		}
	}
	return Verdict{true, ""}
}

func checkLoadBalancer(snap *state.Snapshot) Verdict {
	if !snap.LoadBalancerIsReachable {
		return Verdict{false, "failed to reach external load balancer"}
	}
	return Verdict{true, ""}
}

func baseFitness(q Qualifier, snap *state.Snapshot) Verdict {
	switch q {
	case QPostgres:
		return checkPostgres(snap)
	case QValkey:
		return checkValkey(snap)
	case QSystemd:
		return checkSystemd(snap)
	case QVRRP:
		return checkVRRP(snap)
	case QLoadBalancer:
		return checkLoadBalancer(snap)
	default:
		return Verdict{false, fmt.Sprintf("unknown qualifier: %s", q)}
	}
}

func fitForPrimary(snap *state.Snapshot) Verdict {
	if !snap.OwnsVIP && snap.VRRPRole != "MASTER" {
		return Verdict{false, "not assigned the vrrp vip"}
	}
	if snap.Postgres.Role != string(PostgresPrimary) {
		return Verdict{false, "postgres role is not primary"}
	}
	if snap.Valkey.Role != string(ValkeyPrimary) {
		return Verdict{false, "valkey role is not primary"}
	}
	return Verdict{true, ""}
}

// Standby needs to be all about replication health. A healthy standby must have healthy replication.
// A healthy standby with healthy replication has an active primary. If it doesn't have an active primary,
// then the healthy standby may potentially become the new primary.
func fitForStandby(snap *state.Snapshot) Verdict {
	if snap.Postgres.Role != string(PostgresReplica) {
		return Verdict{false, "postgres role is not replica"}
	}
	if v := pgReplicationHealthy(snap); !v.OK {
		return v
	}
	if snap.Valkey.Role != string(ValkeyReplica) {
		return Verdict{false, "valkey role is not replica"}
	}
	if v := vkReplicationHealthy(snap); !v.OK {
		return v
	}
	return Verdict{true, ""}
}

func pgReplicationHealthy(snap *state.Snapshot) Verdict {
	switch snap.Postgres.Role {
	case string(PostgresPrimary):
		repOK := 0
		for _, r := range snap.Postgres.Replicas {
			if r.LagKnown && r.LagBytes < int64(PostgresLagTheshold) {
				repOK++
			}
			if !r.LagKnown && r.State == "streaming" {
				repOK++
			}
		}
		if repOK > 0 {
			return Verdict{true, ""}
		}
		return Verdict{false, "no suitable replica available"}

	case string(PostgresReplica):
		if !snap.Postgres.StreamingActive {
			return Verdict{false, "no active connection to primary"}
		}
		if snap.Postgres.LagKnown && snap.Postgres.LagBytes > int64(PostgresLagTheshold) {
			return Verdict{false, "replication lag exceeds threshold of 100MB"}
		}
		return Verdict{true, ""}

	default:
		return Verdict{false, "unable to determine postgres role"}
	}
}

func vkReplicationHealthy(snap *state.Snapshot) Verdict {
	return Verdict{true, ""}
}
