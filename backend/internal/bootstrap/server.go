package bootstrap

// Orchestrator daemon entry point.
//
// Reconciles local PG/Valkey roles with VRRP state and peer health.
// The keepalived track_script reads /run/keepalived/role; this daemon
// writes "primary" to it only when local services are fully promoted
// and the load balancer is reachable.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/EHLO1/keel/backend/internal/config"
)

// ─── Domain types ─────────────────────────────────────────────────────────────

type VrrpState string
type PgRole string
type ValkeyRole string

const (
	VrrpMaster  VrrpState = "MASTER"
	VrrpBackup  VrrpState = "BACKUP"
	VrrpFault   VrrpState = "FAULT"
	VrrpUnknown VrrpState = "UNKNOWN"

	PgPrimary PgRole = "primary"
	PgReplica PgRole = "replica"
	PgUnknown PgRole = "unknown"

	ValkeyMaster  ValkeyRole = "master"
	ValkeyReplica ValkeyRole = "replica"
	ValkeyUnknown ValkeyRole = "unknown"
)

type Snapshot struct {
	Vrrp              VrrpState  `json:"vrrp"`
	LocalPg           PgRole     `json:"local_pg"`
	LocalValkey       ValkeyRole `json:"local_valkey"`
	ValkeyMasterHost  string     `json:"valkey_master_host"`
	PeerWgReachable   bool       `json:"peer_wg_reachable"`
	PeerPhysReachable bool       `json:"peer_phys_reachable"`
	PeerPgRole        PgRole     `json:"peer_pg_role"`
	PeerValkeyRole    ValkeyRole `json:"peer_valkey_role"`
	WgHandshakeAge    float64    `json:"wg_handshake_age_sec"`
	LBReachable       bool       `json:"lb_reachable"`
	Maintenance       bool       `json:"maintenance"`
}

type DesiredState struct {
	Pg                PgRole     `json:"pg"`
	Valkey            ValkeyRole `json:"valkey"`
	ValkeyReplicateOf *HostPort  `json:"valkey_replicate_of,omitempty"`
	StateFile         string     `json:"state_file,omitempty"`
	Rationale         string     `json:"rationale"`
}

type HostPort struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ─── Observation ─────────────────────────────────────────────────────────────

func readVrrpState(cfg *config.Config) VrrpState {
	b, err := os.ReadFile(cfg.VRRPStateFile)
	if err != nil {
		return VrrpUnknown
	}
	s := VrrpState(strings.TrimSpace(string(b)))
	switch s {
	case VrrpMaster, VrrpBackup, VrrpFault:
		return s
	}
	return VrrpUnknown
}

func readLocalPgRole(ctx context.Context, cfg *config.Config) PgRole {
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	conn, err := pgx.Connect(cctx, cfg.PGLocalDSN)
	if err != nil {
		return PgUnknown
	}
	defer conn.Close(cctx)
	var inRecovery bool
	if err := conn.QueryRow(cctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return PgUnknown
	}
	if inRecovery {
		return PgReplica
	}
	return PgPrimary
}

func readValkeyInfo(ctx context.Context, cfg *config.Config, addr string) (ValkeyRole, string) {
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	c := redis.NewClient(&redis.Options{
		Addr:        addr,
		DialTimeout: cfg.ProbeTimeout,
		ReadTimeout: cfg.ProbeTimeout,
	})
	defer c.Close()
	out, err := c.Info(cctx, "replication").Result()
	if err != nil {
		return ValkeyUnknown, ""
	}
	role, masterHost := "", ""
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch k {
		case "role":
			role = v
		case "master_host":
			masterHost = v
		}
	}
	switch role {
	case "master":
		return ValkeyMaster, ""
	case "slave":
		return ValkeyReplica, masterHost
	}
	return ValkeyUnknown, ""
}

func probePeerPg(ctx context.Context, cfg *config.Config) (bool, PgRole) {
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	conn, err := pgx.Connect(cctx, cfg.PGPeerDSN())
	if err != nil {
		return false, PgUnknown
	}
	defer conn.Close(cctx)
	var inRecovery bool
	if err := conn.QueryRow(cctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return false, PgUnknown
	}
	if inRecovery {
		return true, PgReplica
	}
	return true, PgPrimary
}

func probeHTTP(ctx context.Context, cfg *config.Config, url string) bool {
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func wgHandshakeAge(cfg *config.Config) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ProbeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "wg", "show", cfg.WGInterface, "latest-handshakes").Output()
	if err != nil {
		return -1
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return -1
	}
	fields := strings.Fields(line)
	ts, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil || ts == 0 {
		return -1
	}
	return time.Since(time.Unix(ts, 0)).Seconds()
}

func collectSnapshot(ctx context.Context, cfg *config.Config) Snapshot {
	var (
		wg                             sync.WaitGroup
		vrrp                           VrrpState
		localPg, peerPgRole            PgRole
		localValkey, peerValkey        ValkeyRole
		valkeyMaster                   string
		peerPgReach, peerPhysReach, lb bool
		wgAge                          float64
		maintenance                    bool
	)

	probes := []func(){
		func() { vrrp = readVrrpState(cfg) },
		func() { localPg = readLocalPgRole(ctx, cfg) },
		func() { localValkey, valkeyMaster = readValkeyInfo(ctx, cfg, cfg.ValkeyLocalAddr) },
		func() { peerPgReach, peerPgRole = probePeerPg(ctx, cfg) },
		func() { peerValkey, _ = readValkeyInfo(ctx, cfg, cfg.ValkeyPeerAddr()) },
		func() { peerPhysReach = probeHTTP(ctx, cfg, cfg.PeerQueueHealthURL()) },
		func() { lb = probeHTTP(ctx, cfg, cfg.LBHealthURL) },
		func() { wgAge = wgHandshakeAge(cfg) },
		func() { _, err := os.Stat(cfg.MaintenanceFile); maintenance = err == nil },
	}
	for _, p := range probes {
		wg.Add(1)
		go func(fn func()) { defer wg.Done(); fn() }(p)
	}
	wg.Wait()

	peerWgReach := peerPgReach || peerValkey != ValkeyUnknown
	return Snapshot{
		Vrrp: vrrp, LocalPg: localPg, LocalValkey: localValkey,
		ValkeyMasterHost:  valkeyMaster,
		PeerWgReachable:   peerWgReach,
		PeerPhysReachable: peerPhysReach,
		PeerPgRole:        peerPgRole, PeerValkeyRole: peerValkey,
		WgHandshakeAge: wgAge, LBReachable: lb, Maintenance: maintenance,
	}
}

// ─── Pure decision logic ─────────────────────────────────────────────────────

func computeDesiredState(cfg *config.Config, s Snapshot, peerDownStrikes int) DesiredState {
	if s.Maintenance {
		return DesiredState{
			Pg: s.LocalPg, Valkey: s.LocalValkey,
			Rationale: "maintenance flag set; holding current state",
		}
	}

	peerTrulyDown := !s.PeerWgReachable && !s.PeerPhysReachable &&
		peerDownStrikes >= cfg.PeerDownHysteresis
	splitBrainRisk := !s.PeerWgReachable && s.PeerPhysReachable

	switch s.Vrrp {
	case VrrpMaster:
		if splitBrainRisk {
			return DesiredState{
				Pg: s.LocalPg, Valkey: s.LocalValkey,
				Rationale: "MASTER but peer reachable on phys net; split-brain risk, holding",
			}
		}
		stateFile := ""
		if s.LBReachable {
			stateFile = "primary"
		}
		rationale := "MASTER with peer reachable; promoting (clean takeover)"
		if peerTrulyDown || !s.PeerWgReachable {
			rationale = "MASTER and peer down; promoting locally"
		}
		return DesiredState{
			Pg: PgPrimary, Valkey: ValkeyMaster,
			StateFile: stateFile, Rationale: rationale,
		}

	case VrrpBackup:
		if s.PeerWgReachable {
			return DesiredState{
				Pg: PgReplica, Valkey: ValkeyReplica,
				ValkeyReplicateOf: &HostPort{Host: cfg.PeerWGIP, Port: cfg.ValkeyPeerPort},
				Rationale:         "BACKUP and peer reachable; following peer",
			}
		}
		return DesiredState{
			Pg: s.LocalPg, Valkey: s.LocalValkey,
			Rationale: "BACKUP but peer unreachable; holding (waiting for VRRP)",
		}
	}

	return DesiredState{
		Pg: s.LocalPg, Valkey: s.LocalValkey,
		Rationale: fmt.Sprintf("VRRP=%s; taking no action", s.Vrrp),
	}
}

// ─── Action layer (idempotent) ───────────────────────────────────────────────

func ensurePgPrimary(ctx context.Context, cfg *config.Config, current PgRole) error {
	if current == PgPrimary {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, cfg.ActionTimeout)
	defer cancel()
	return exec.CommandContext(cctx, cfg.RepmgrBinary, "-f", cfg.RepmgrConfig,
		"standby", "promote", "--log-to-file").Run()
}

func ensurePgReplica(ctx context.Context, cfg *config.Config, current PgRole) error {
	if current == PgReplica {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, cfg.ActionTimeout)
	defer cancel()
	return exec.CommandContext(cctx, cfg.RepmgrBinary, "-f", cfg.RepmgrConfig,
		"standby", "follow", "--log-to-file").Run()
}

func ensureValkeyMaster(ctx context.Context, cfg *config.Config, current ValkeyRole) error {
	if current == ValkeyMaster {
		return nil
	}
	c := redis.NewClient(&redis.Options{Addr: cfg.ValkeyLocalAddr, DialTimeout: cfg.ProbeTimeout})
	defer c.Close()
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	return c.SlaveOf(cctx, "NO", "ONE").Err()
}

func ensureValkeyReplica(ctx context.Context, cfg *config.Config,
	current ValkeyRole, currentMaster string, target HostPort) error {
	if current == ValkeyReplica && currentMaster == target.Host {
		return nil
	}
	c := redis.NewClient(&redis.Options{Addr: cfg.ValkeyLocalAddr, DialTimeout: cfg.ProbeTimeout})
	defer c.Close()
	cctx, cancel := context.WithTimeout(ctx, cfg.ProbeTimeout)
	defer cancel()
	return c.SlaveOf(cctx, target.Host, strconv.Itoa(target.Port)).Err()
}

func writeStateFile(path, value string) error {
	if value == "" {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(value), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func apply(ctx context.Context, cfg *config.Config, d DesiredState, s Snapshot, log *slog.Logger) {
	if d.Pg == PgPrimary {
		if err := ensurePgPrimary(ctx, cfg, s.LocalPg); err != nil {
			log.Error("ensure_pg_primary_failed", "err", err)
		}
	} else if d.Pg == PgReplica {
		if err := ensurePgReplica(ctx, cfg, s.LocalPg); err != nil {
			log.Error("ensure_pg_replica_failed", "err", err)
		}
	}
	if d.Valkey == ValkeyMaster {
		if err := ensureValkeyMaster(ctx, cfg, s.LocalValkey); err != nil {
			log.Error("ensure_valkey_master_failed", "err", err)
		}
	} else if d.Valkey == ValkeyReplica && d.ValkeyReplicateOf != nil {
		if err := ensureValkeyReplica(ctx, cfg, s.LocalValkey,
			s.ValkeyMasterHost, *d.ValkeyReplicateOf); err != nil {
			log.Error("ensure_valkey_replica_failed", "err", err)
		}
	}
	if err := writeStateFile(cfg.StateFile, d.StateFile); err != nil {
		log.Error("write_state_file_failed", "err", err)
	}
}

// ─── Main ────────────────────────────────────────────────────────────────────

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	return slog.LevelInfo
}

func newLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}
	if cfg.LogJSON {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func main() {
	envFile := flag.String("env-file", "/etc/orchestrator/.env",
		"Path to .env file; empty disables file loading")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	log := newLogger(cfg)
	maskedJSON, _ := json.Marshal(cfg.MaskSensitive())
	log.Info("starting", "config", json.RawMessage(maskedJSON))

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	tick := time.NewTicker(cfg.TickInterval)
	defer tick.Stop()

	peerDownStrikes := 0

	runOnce := func() {
		snap := collectSnapshot(ctx, cfg)
		if !snap.PeerWgReachable && !snap.PeerPhysReachable {
			peerDownStrikes++
		} else {
			peerDownStrikes = 0
		}
		desired := computeDesiredState(cfg, snap, peerDownStrikes)

		snapJSON, _ := json.Marshal(snap)
		desJSON, _ := json.Marshal(desired)
		log.Info("tick",
			"snapshot", json.RawMessage(snapJSON),
			"desired", json.RawMessage(desJSON))

		apply(ctx, cfg, desired, snap, log)
	}

	runOnce()
	for {
		select {
		case <-ctx.Done():
			log.Info("shutdown")
			return
		case <-tick.C:
			runOnce()
		}
	}
}
