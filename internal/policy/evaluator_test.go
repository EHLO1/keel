package policy

import (
	"testing"

	"github.com/EHLO1/keel/internal/state"
)

func TestEvaluator(t *testing.T) {
	eval, err := NewEvaluator("10.0.0.2", 6379)
	if err != nil {
		t.Fatalf("failed to create evaluator: %v", err)
	}

	// Maintenance Mode
	snap := &state.Snapshot{
		MaintenanceMode: true,
	}
	desired := eval.Evaluate(snap)
	if desired.Postgres != PostgresUnknown || desired.Valkey != ValkeyUnknown {
		t.Errorf("expected Unknown in maintenance mode, got Postgres=%s Valkey=%s", desired.Postgres, desired.Valkey)
	}

	// Active Node (owns VIP)
	snap = &state.Snapshot{
		OwnsVIP:  true,
		VRRPRole: "BACKUP", // OwnsVIP takes priority
	}
	desired = eval.Evaluate(snap)
	if desired.Postgres != PostgresPrimary || desired.Valkey != ValkeyMaster {
		t.Errorf("expected primary/master when owning VIP, got Postgres=%s Valkey=%s", desired.Postgres, desired.Valkey)
	}

	// Standby Node (does not own VIP, BACKUP role)
	snap = &state.Snapshot{
		OwnsVIP:  false,
		VRRPRole: "BACKUP",
	}
	desired = eval.Evaluate(snap)
	if desired.Postgres != PostgresReplica || desired.Valkey != ValkeyReplica {
		t.Errorf("expected replica when standby, got Postgres=%s Valkey=%s", desired.Postgres, desired.Valkey)
	}
	if desired.ValkeyReplicaOf == nil || desired.ValkeyReplicaOf.Host != "10.0.0.2" || desired.ValkeyReplicaOf.Port != 6379 {
		t.Errorf("expected valkey replica of 10.0.0.2:6379, got %v", desired.ValkeyReplicaOf)
	}
}
