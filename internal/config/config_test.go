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
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL")
	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TickInterval != 3*time.Second {
		t.Errorf("default TickInterval: got %v", cfg.TickInterval)
	}
	if cfg.WireguardInterface != "wg0" {
		t.Errorf("default WireguardInterface: got %q", cfg.WireguardInterface)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel: got %q", cfg.LogLevel)
	}
	if !cfg.LogJSON {
		t.Errorf("default LogJSON should be true")
	}
}

func TestEnvFileLoads(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL", "TICK_INTERVAL")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
# This is the canonical orchestrator config
WIREGUARD_PEER_IP=10.0.0.2
REAL_PEER_IP=192.168.1.2
LOAD_BALANCER_HEALTH_URL="http://lb.example/healthz"
TICK_INTERVAL=5s
LOG_LEVEL=debug
`)

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WireguardPeerIP != "10.0.0.2" {
		t.Errorf("WireguardPeerIP: got %q", cfg.WireguardPeerIP)
	}
	if cfg.LoadBalancerHealthURL != "http://lb.example/healthz" {
		t.Errorf("LoadBalancerHealthURL: got %q", cfg.LoadBalancerHealthURL)
	}
	if cfg.TickInterval != 5*time.Second {
		t.Errorf("TickInterval: got %v", cfg.TickInterval)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q (toLower applied?)", cfg.LogLevel)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
WIREGUARD_PEER_IP=from_file
REAL_PEER_IP=192.168.1.2
LOAD_BALANCER_HEALTH_URL=http://lb.example/healthz
`)

	t.Setenv("WIREGUARD_PEER_IP", "from_env")

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WireguardPeerIP != "from_env" {
		t.Errorf("real env should override file: got %q", cfg.WireguardPeerIP)
	}
}

func TestToLowerOption(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL", "LOG_LEVEL")
	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")
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
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL", "POSTGRES_PASSWORD", "POSTGRES_PASSWORD_FILE")

	dir := t.TempDir()
	secretPath := writeFile(t, dir, "pg_pass", "hunter2\n")

	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")
	t.Setenv("POSTGRES_PASSWORD_FILE", secretPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PostgresPassword != "hunter2" {
		t.Errorf("file-based secret not loaded: got %q", cfg.PostgresPassword)
	}
}

func TestValidateRequiresPeerIPs(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL")

	if _, err := Load(""); err == nil {
		t.Errorf("expected error when required vars missing")
	}
}

func TestMaskSensitive(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL", "POSTGRES_PASSWORD")
	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")
	t.Setenv("POSTGRES_PASSWORD", "secret")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	masked := cfg.MaskSensitive()
	if masked["POSTGRES_PASSWORD"] != "****" {
		t.Errorf("POSTGRES_PASSWORD should be masked: %v", masked["POSTGRES_PASSWORD"])
	}
	if masked["WIREGUARD_PEER_IP"] != "10.0.0.2" {
		t.Errorf("WIREGUARD_PEER_IP should not be masked: %v", masked["WIREGUARD_PEER_IP"])
	}
}

func TestDerivedAccessors(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL")
	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := "http://192.168.1.2:9999/queue-health"
	if got := cfg.PeerQueueHealthURL(); got != want {
		t.Errorf("PeerQueueHealthURL: got %q, want %q", got, want)
	}
	if got := cfg.ValkeyPeerAddr(); got != "10.0.0.2:6379" {
		t.Errorf("ValkeyPeerAddr: got %q", got)
	}
}

func TestEnvFileCommentsAndQuotes(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL", "VALKEY_HOST")

	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `
# Full-line comment
WIREGUARD_PEER_IP=10.0.0.2  # trailing comment ignored
REAL_PEER_IP='192.168.1.2'
LOAD_BALANCER_HEALTH_URL="http://lb.example/healthz"
VALKEY_HOST="127.0.0.1"
`)

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WireguardPeerIP != "10.0.0.2" {
		t.Errorf("trailing comment: got %q", cfg.WireguardPeerIP)
	}
	if cfg.RealPeerIP != "192.168.1.2" {
		t.Errorf("single-quoted: got %q", cfg.RealPeerIP)
	}
	if cfg.ValkeyHost != "127.0.0.1" {
		t.Errorf("double-quoted: got %q", cfg.ValkeyHost)
	}
}

func TestEnvFileMissingIsOK(t *testing.T) {
	clearEnv(t, "WIREGUARD_PEER_IP", "REAL_PEER_IP", "LOAD_BALANCER_HEALTH_URL")
	t.Setenv("WIREGUARD_PEER_IP", "10.0.0.2")
	t.Setenv("REAL_PEER_IP", "192.168.1.2")
	t.Setenv("LOAD_BALANCER_HEALTH_URL", "http://lb.example/healthz")

	if _, err := Load("/nonexistent/.env"); err != nil {
		t.Errorf("missing file should not be a fatal error: %v", err)
	}
}
