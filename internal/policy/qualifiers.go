package policy

import (
	"github.com/EHLO1/keel/internal/state"
)

// TODO: Add resilience against minor network hitches
func meetsBaseQualifiers(snap *state.Snapshot) bool {
	if snap.MaintenanceMode {
		return false
	}
	for _, svc := range snap.Systemd.Services {
		if svc.ActiveState != "running" && svc.SubState != "enabled" {
			return false
		}
	}
	if snap.VRRPRole == "FAULT" || snap.VRRPRole == "UNKNOWN" {
		return false
	}
	if !snap.LoadBalancerIsReachable {
		return false
	}
	if !snap.Postgres.Reachable {
		return false
	}
	if !snap.Valkey.Reachable {
		return false
	}

	return true
}
