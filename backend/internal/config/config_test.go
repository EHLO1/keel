package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL")
	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TickInterval != 3*time.Second {
		t.Errorf("default TickInterval: got %v", cfg.TickInterval)
	}
	if cfg.WGInterface != "wg0" {
		t.Errorf("default WGInterface: got %q", cfg.WGInterface)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel: got %q", cfg.LogLevel)
	}
	if !cfg.LogJSON {
		t.Errorf("default LogJSON should be true")
	}
}

func TestEnvFileLoads(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL", "TICK_INTERVAL")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
# This is the canonical orchestrator config
PEER_WG_IP=10.0.0.2
PEER_PHYS_IP=192.168.1.2
LB_HEALTH_URL="http://lb.example/healthz"
TICK_INTERVAL=5s
LOG_LEVEL=debug
`)

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PeerWGIP != "10.0.0.2" {
		t.Errorf("PeerWGIP: got %q", cfg.PeerWGIP)
	}
	if cfg.LBHealthURL != "http://lb.example/healthz" {
		t.Errorf("LBHealthURL: got %q", cfg.LBHealthURL)
	}
	if cfg.TickInterval != 5*time.Second {
		t.Errorf("TickInterval: got %v", cfg.TickInterval)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q (toLower applied?)", cfg.LogLevel)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
PEER_WG_IP=from_file
PEER_PHYS_IP=192.168.1.2
LB_HEALTH_URL=http://lb.example/healthz
`)

	t.Setenv("PEER_WG_IP", "from_env")

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PeerWGIP != "from_env" {
		t.Errorf("real env should override file: got %q", cfg.PeerWGIP)
	}
}

func TestToLowerOption(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL", "LOG_LEVEL")
	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")
	t.Setenv("LOG_LEVEL", "DEBUG")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("toLower should lowercase: got %q", cfg.LogLevel)
	}
}

func TestFileBasedSecret(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL", "PG_LOCAL_DSN", "PG_LOCAL_DSN_FILE")

	dir := t.TempDir()
	secretPath := writeFile(t, dir, "pg_dsn", "host=10.99.99.99 user=secret password=hunter2\n")

	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")
	t.Setenv("PG_LOCAL_DSN_FILE", secretPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PGLocalDSN != "host=10.99.99.99 user=secret password=hunter2" {
		t.Errorf("file-based secret not loaded: got %q", cfg.PGLocalDSN)
	}
}

func TestValidateRequiresPeerIPs(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL")

	if _, err := Load(""); err == nil {
		t.Errorf("expected error when required vars missing")
	}
}

func TestMaskSensitive(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL", "PG_LOCAL_DSN")
	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")
	t.Setenv("PG_LOCAL_DSN", "host=somewhere password=secret")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	masked := cfg.MaskSensitive()
	if masked["PG_LOCAL_DSN"] != "****" {
		t.Errorf("PG_LOCAL_DSN should be masked: %v", masked["PG_LOCAL_DSN"])
	}
	if masked["PEER_WG_IP"] != "10.0.0.2" {
		t.Errorf("PEER_WG_IP should not be masked: %v", masked["PEER_WG_IP"])
	}
}

func TestDerivedAccessors(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL")
	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := "http://192.168.1.2:8080/queue-health"
	if got := cfg.PeerQueueHealthURL(); got != want {
		t.Errorf("PeerQueueHealthURL: got %q, want %q", got, want)
	}
	if got := cfg.ValkeyPeerAddr(); got != "10.0.0.2:6379" {
		t.Errorf("ValkeyPeerAddr: got %q", got)
	}
}

func TestEnvFileCommentsAndQuotes(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL", "VALKEY_LOCAL_ADDR")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
# Full-line comment
PEER_WG_IP=10.0.0.2  # trailing comment ignored
PEER_PHYS_IP='192.168.1.2'
LB_HEALTH_URL="http://lb.example/healthz"
VALKEY_LOCAL_ADDR="127.0.0.1:6379"
`)

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PeerWGIP != "10.0.0.2" {
		t.Errorf("trailing comment: got %q", cfg.PeerWGIP)
	}
	if cfg.PeerPhysIP != "192.168.1.2" {
		t.Errorf("single-quoted: got %q", cfg.PeerPhysIP)
	}
	if cfg.ValkeyLocalAddr != "127.0.0.1:6379" {
		t.Errorf("double-quoted: got %q", cfg.ValkeyLocalAddr)
	}
}

func TestEnvFileMissingIsOK(t *testing.T) {
	clearEnv(t, "PEER_WG_IP", "PEER_PHYS_IP", "LB_HEALTH_URL")
	t.Setenv("PEER_WG_IP", "10.0.0.2")
	t.Setenv("PEER_PHYS_IP", "192.168.1.2")
	t.Setenv("LB_HEALTH_URL", "http://lb.example/healthz")

	if _, err := Load("/nonexistent/.env"); err != nil {
		t.Errorf("missing file should not be a fatal error: %v", err)
	}
}
