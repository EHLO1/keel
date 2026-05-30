package policy

import (
	"fmt"

	"github.com/EHLO1/keel/internal/state"
)

// TODO: Add resilience against minor network hitches
func meetsBaseQualifiers(snap *state.Snapshot) Verdict {
	if snap.MaintenanceMode {
		return Verdict{false, "maintenance mode is active"}
	}
	for _, svc := range snap.Systemd.Services {
		if svc.ActiveState != "running" && svc.SubState != "enabled" {
			return Verdict{false, fmt.Sprintf("service is not running or is not enabled: %s", svc)}
		}
	}
	if snap.VRRPRole == "FAULT" || snap.VRRPRole == "UNKNOWN" {
		return Verdict{false, "vrrp is not ready"}
	}
	if !snap.LoadBalancerIsReachable {
		return Verdict{false, "failed to reach external load balancer"}
	}
	if !snap.Postgres.Reachable {
		return Verdict{false, "local postgres state is unknown"}
	}
	if !snap.Valkey.Reachable {
		return Verdict{false, "local valkey or redis state is unknown"}
	}
	return Verdict{true, ""}
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
