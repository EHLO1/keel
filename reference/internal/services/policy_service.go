package services

import (
	"github.com/EHLO1/keel/backend/internal/config"
)

type PolicyService struct {
	cfg *config.Config
}

// DesiredState is the output of policy evaluation: what each subsystem
// should be doing, given the current snapshot.
type DesiredState struct {
	Pg                PgRole
	Valkey            ValkeyRole
	ValkeyReplicateOf *HostPort
	StateFile         string // "primary" or "" for absent
	Rationale         string
}

type PgRole string
type ValkeyRole string
type HostPort struct {
	Host string
	Port int
}

const (
	PgPrimary PgRole = "primary"
	PgReplica PgRole = "replica"
	PgUnknown PgRole = "unknown"

	ValkeyMaster  ValkeyRole = "master"
	ValkeyReplica ValkeyRole = "replica"
	ValkeyUnknown ValkeyRole = "unknown"
)

func NewPolicyService(cfg *config.Config) *PolicyService {
	return &PolicyService{cfg: cfg}
}

func (p *PolicyService) Evaluate(snapshot *Snapshot) DesiredState {
	d := DesiredState{}
	return d
}
