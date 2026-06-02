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

func pgReplicationHealthy(snap *state.Snapshot, pgLagThreshold int64) Verdict {
	switch snap.Postgres.Role {
	case "primary":
		repOK := 0
		for _, r := range snap.Postgres.Replicas {
			if r.LagKnown && r.LagBytes < pgLagThreshold {
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

	case "replica":
		if !snap.Postgres.StreamingActive {
			return Verdict{false, "no active connection to primary"}
		}
		if snap.Postgres.LagKnown && snap.Postgres.LagBytes > pgLagThreshold {
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
