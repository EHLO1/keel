package actor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EHLO1/keel/internal/adapter/docker"
	"github.com/EHLO1/keel/internal/adapter/filesystem"
	"github.com/EHLO1/keel/internal/adapter/postgres"
	"github.com/EHLO1/keel/internal/adapter/systemd"
	"github.com/EHLO1/keel/internal/adapter/valkey"
	"github.com/EHLO1/keel/internal/config"
	"github.com/EHLO1/keel/internal/policy"
	"github.com/EHLO1/keel/internal/state"
)

type Dependencies struct {
	Config *config.Config
	ST     *state.Service
	PG     *postgres.Client
	VK     *valkey.Client
	DR     *docker.Client
	Sys    systemd.Client
	SS     *filesystem.StandbySignal
	SF     *filesystem.StateFile
	Log    *slog.Logger
}

type ActorEnforcer struct {
	cfg *config.Config
	st  *state.Service
	pg  *postgres.Client
	vk  *valkey.Client
	dr  *docker.Client
	sys systemd.Client
	ss  *filesystem.StandbySignal
	sf  *filesystem.StateFile
	log *slog.Logger
}

func NewEnforcer(deps Dependencies) (*ActorEnforcer, error) {
	return &ActorEnforcer{
		cfg: deps.Config,
		st:  deps.ST,
		pg:  deps.PG,
		vk:  deps.VK,
		dr:  deps.DR,
		sys: deps.Sys,
		ss:  deps.SS,
		sf:  deps.SF,
		log: deps.Log,
	}, nil
}

func (e *ActorEnforcer) Apply(ctx context.Context, desired *policy.DesiredState) error {
	curr := e.st.Current()

	// ── 1. Reconcile State File ─────────────────────
	if curr.MaintenanceMode {
		e.log.Info("maintenance mode active, setting state file to unhealthy")
		if err := e.sf.SetUnhealthy(); err != nil {
			e.log.Error("failed to set state file to unhealthy", "error", err)
		}
	} else {
		if err := e.sf.SetHealthy(); err != nil {
			e.log.Error("failed to set state file healthy", "error", err)
		}
	}

	// ── 2. Reconcile PostgreSQL ──────────────────────────────────────────────
	if desired.Postgres == policy.PostgresPrimary {
		if curr.Postgres.Role == string(postgres.RoleReplica) || curr.PostgresInStandby {
			e.log.Info("reconciling postgres: promoting to primary")

			if err := e.pg.Promote(ctx); err != nil {
				return fmt.Errorf("failed to promote postgres to primary: %w", err)
			}

			pgVolPath, err := e.dr.GetVolumeMountpoint(ctx)
			if err != nil {
				e.log.Error("failed to get postgres volume mountpoint to remove standby.signal", "error", err)
			} else {
				if err := e.ss.RemoveStandby(pgVolPath); err != nil {
					e.log.Error("failed to remove standby.signal", "error", err)
				}
			}
		}
	} else if desired.Postgres == policy.PostgresReplica {
		if curr.Postgres.Role == string(postgres.RolePrimary) || !curr.PostgresInStandby {
			e.log.Info("reconciling postgres: demoting to replica")

			pgVolPath, err := e.dr.GetVolumeMountpoint(ctx)
			if err != nil {
				return fmt.Errorf("failed to get volume mountpoint for demotion: %w", err)
			}
			if err := e.ss.SetStandby(pgVolPath); err != nil {
				return fmt.Errorf("failed to write standby.signal: %w", err)
			}

			if err := e.pg.Demote(ctx, e.cfg.WireguardPeerIP, e.cfg.PostgresPort, e.cfg.PostgresUser, e.cfg.PostgresPassword); err != nil {
				return fmt.Errorf("failed to write primary_conninfo for demotion: %w", err)
			}

			if err := e.dr.RestartPostgresContainer(ctx); err != nil {
				return fmt.Errorf("failed to restart postgres container: %w", err)
			}
		}
	}

	// ── 3. Reconcile Valkey ─────────────────────────────────────────────────
	if desired.Valkey == policy.ValkeyMaster {
		if curr.Valkey.Role != "master" {
			e.log.Info("reconciling valkey: promoting to master")
			if err := e.vk.PromoteToMaster(ctx); err != nil {
				return fmt.Errorf("failed to promote valkey to master: %w", err)
			}
		}
	} else if desired.Valkey == policy.ValkeyReplica && desired.ValkeyReplicaOf != nil {
		if curr.Valkey.Role != "replica" && curr.Valkey.Role != "slave" {
			e.log.Info("reconciling valkey: setting replica of", "host", desired.ValkeyReplicaOf.Host, "port", desired.ValkeyReplicaOf.Port)
			if err := e.vk.MakeReplicaOf(ctx, desired.ValkeyReplicaOf.Host, desired.ValkeyReplicaOf.Port); err != nil {
				return fmt.Errorf("failed to make valkey replica of %s:%d: %w", desired.ValkeyReplicaOf.Host, desired.ValkeyReplicaOf.Port, err)
			}
		}
	}

	return nil
}
