package policy

import (
	"github.com/EHLO1/keel/internal/adapter/systemd"
	"github.com/EHLO1/keel/internal/state"
)

func systemdServiceCheck(snap *state.Snapshot) (bool, error) {
	wgSvc := &snap.Systemd.Services[].Name
	
}

func meetsBaseQualifiers(snap *state.Snapshot) bool {
	if !snap.MaintenanceMode &&
		!snap.Systemd.Services
}
