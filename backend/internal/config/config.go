// Package config holds all orchestrator configuration.
//
// Fields tagged with `env` are loaded from environment variables.
// Fields with `options:"file"` support secrets via the _FILE suffix
// (matching the Arcane / Docker secrets convention).
//
// Config sources, in precedence order (highest first):
//  1. Real environment variables (set by systemd, shell, etc.)
//  2. .env file (default: /etc/orchestrator/.env, override with --env-file)
//  3. `default` tag on each struct field
package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Runtime configuration for the orchestrator daemon.
type Config struct {
	// ── VRRP topology ────────────────────────────────────────────────────────
	// IP of the local node on the wg0 tunnel; required.
	WireguardIP string `env:"WIREGUARD_IP" default:""`
	// Real/Physical IP of the local node on the physical network; required.
	RealIP string `env:"REAL_IP" default:""`
	// IP of the peer node on the wg0 tunnel; required.
	WireguardPeerIP string `env:"WIREGUARD_PEER_IP" default:""`
	// Real/Physical IP of the peer node on the physical network; required.
	RealPeerIP string `env:"REAL_PEER_IP" default:""`

	// ── Reachability probes ──────────────────────────────────────────────────
	// Local Hostname.
	PeerHostname string `env:"HOSTNAME" default:"" options:"toLower"`
	// Health endpoint for the load balancer (used to gate "primary" advert).
	LoadBalancerHealthURL string `env:"LOAD_BALANCER_HEALTH_URL" default:""`
	// Port on the peer where the queue-health autoscaler listens.
	PeerQueueHealthPort int `env:"PEER_QUEUE_HEALTH_PORT" default:"9999"`
	// Path on the peer's queue-health endpoint.
	PeerQueueHealthPath string `env:"PEER_QUEUE_HEALTH_PATH" default:"/queue-health"`

	// ── PostgreSQL ───────────────────────────────────────────────────────────
	// Repmgr Connection Info
	PostgresReplicationDB   string `env:"POSTGRES_REPLICATION_DB" default:"repmgr"`
	PostgresReplicationUser string `env:"POSTGRES_REPLICATION_USER" default:"repmgr"`
	PostgresPort            int    `env:"POSTGRES_PORT" default:"5432"`
	// DSN for the Postgres instances (orchestrator only reads role).
	PostgresReplicationDSN string `env:"POSTGRES_REPLICATION_DSN" default:"host=%s port=%d user=%s dbname=%s connect_timeout=2" options:"file"`

	// ── Valkey ───────────────────────────────────────────────────────────────
	ValkeyPort     int    `env:"VALKEY_PORT" default:"6379"`
	ValkeyPassword string `env:"VALKEY_PASSWORD" default:""`

	// ── WireGuard ────────────────────────────────────────────────────────────
	WireguardInterface      string        `env:"WIREGUARD_INTERFACE" default:"wg0"`
	WireguardHandshakeStale time.Duration `env:"WIREGUARD_HANDSHAKE_STALE" default:"75s"`

	// ── State files ──────────────────────────────────────────────────────────
	// File the track_script reads to gate the +50 weight.
	StateFile string `env:"STATE_FILE" default:"/run/keepalived/role"`
	// File the keepalived notify_* scripts write the current VRRP state to.
	VRRPStateFile string `env:"VRRP_STATE_FILE" default:"/run/keepalived/vrrp_state"`
	// File the deployment script touches to suspend orchestrator action.
	MaintenanceFile string `env:"MAINTENANCE_FILE" default:"/run/keepalived/maintenance"`

	// ── Loop tuning ──────────────────────────────────────────────────────────
	TickInterval    time.Duration `env:"TICK_INTERVAL" default:"3s"`
	PeerDownStrikes int           `env:"PEER_DOWN_STRIKES" default:"3"`
	ProbeTimeout    time.Duration `env:"PROBE_TIMEOUT" default:"2s"`
	ActionTimeout   time.Duration `env:"ACTION_TIMEOUT" default:"60s"`

	// ── External tools ───────────────────────────────────────────────────────
	RepmgrBinary string `env:"REPMGR_BINARY" default:"/usr/bin/repmgr"`
	RepmgrConfig string `env:"REPMGR_CONFIG" default:"/etc/repmgr.conf"`

	// ── Logging ──────────────────────────────────────────────────────────────
	LogLevel string `env:"LOG_LEVEL" default:"info" options:"toLower"`
	LogJSON  bool   `env:"LOG_JSON" default:"true"`
}

// Load reads the optional .env file at envFilePath (empty = skip), overlays
// real environment variables on top, and resolves the result into a Config.
// Returns an error if required fields are missing or malformed.
func Load(envFilePath string) (*Config, error) {
	if envFilePath != "" {
		if err := loadEnvFile(envFilePath); err != nil {
			// Missing file is fine; malformed file is not.
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read env file %s: %w", envFilePath, err)
			}
			slog.Debug("env file not present, using environment + defaults",
				"path", envFilePath)
		}
	}

	cfg := &Config{}
	loadFromEnv(cfg)
	applyOptions(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate ensures required fields are present and ranges are sensible.
func (c *Config) Validate() error {
	if c.PeerWGIP == "" {
		return fmt.Errorf("PEER_WG_IP is required")
	}
	if c.PeerPhysIP == "" {
		return fmt.Errorf("PEER_PHYS_IP is required")
	}
	if c.LBHealthURL == "" {
		return fmt.Errorf("LB_HEALTH_URL is required")
	}
	if c.TickInterval <= 0 {
		return fmt.Errorf("TICK_INTERVAL must be positive")
	}
	if c.PeerDownHysteresis < 1 {
		return fmt.Errorf("PEER_DOWN_HYSTERESIS must be >= 1")
	}
	return nil
}

// PeerQueueHealthURL is the full URL of the peer's queue-health endpoint.
func (c *Config) PeerQueueHealthURL() string {
	return fmt.Sprintf("http://%s:%d%s",
		c.PeerPhysIP, c.PeerQueueHealthPort, c.PeerQueueHealthPath)
}

// PGPeerDSN renders the peer DSN with the current peer wg0 IP.
func (c *Config) PGPeerDSN() string {
	return fmt.Sprintf(c.PGPeerDSNTemplate, c.PeerWGIP)
}

// ValkeyPeerAddr is the host:port form of the peer Valkey server.
func (c *Config) ValkeyPeerAddr() string {
	return fmt.Sprintf("%s:%d", c.PeerWGIP, c.ValkeyPeerPort)
}

// MaskSensitive returns a copy of the config with sensitive fields redacted.
// Fields tagged with options:"file" are considered sensitive.
func (c *Config) MaskSensitive() map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ft := t.Field(i)
		envTag := ft.Tag.Get("env")
		if envTag == "" {
			envTag = ft.Name
		}
		isSensitive := strings.Contains(ft.Tag.Get("options"), "file")
		if isSensitive {
			s := fmt.Sprintf("%v", field.Interface())
			if s == "" {
				result[envTag] = "(empty)"
			} else {
				result[envTag] = "****"
			}
		} else {
			result[envTag] = field.Interface()
		}
	}
	return result
}

// ─── Loader internals (Arcane-style reflection) ───────────────────────────────

func loadFromEnv(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		setFieldValue(v.Field(i), t.Field(i))
	}
}

func setFieldValue(field reflect.Value, ft reflect.StructField) {
	envTag := ft.Tag.Get("env")
	if envTag == "" {
		return
	}
	defaultValue := ft.Tag.Get("default")
	value := trimQuotes(os.Getenv(envTag))
	if value == "" {
		value = defaultValue
	}
	if !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		if b, err := strconv.ParseBool(value); err == nil {
			field.SetBool(b)
		}
	case reflect.Int, reflect.Int64:
		// time.Duration is an int64 under the hood; check for it specifically.
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			if d, err := time.ParseDuration(value); err == nil {
				field.SetInt(int64(d))
			} else {
				slog.Warn("invalid duration; using zero",
					"field", envTag, "value", value, "err", err)
			}
			return
		}
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(i)
		}
	}
}

func applyOptions(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ft := t.Field(i)
		opts := ft.Tag.Get("options")
		if opts == "" {
			continue
		}
		for _, opt := range strings.Split(opts, ",") {
			switch strings.TrimSpace(opt) {
			case "file":
				resolveFileBased(field, ft)
			case "toLower":
				if field.Kind() == reflect.String {
					field.SetString(strings.ToLower(field.String()))
				}
			case "trimTrailingSlash":
				if field.Kind() == reflect.String {
					field.SetString(strings.TrimRight(field.String(), "/"))
				}
			}
		}
	}
}

// resolveFileBased checks for $VAR_FILE; if present, reads the file and uses
// its contents as the field value. Used for Docker secrets.
func resolveFileBased(field reflect.Value, ft reflect.StructField) {
	if field.Kind() != reflect.String {
		return
	}
	envTag := ft.Tag.Get("env")
	if envTag == "" {
		return
	}
	var path string
	for _, suffix := range []string{"__FILE", "_FILE"} {
		if p := os.Getenv(envTag + suffix); p != "" {
			path = p
			break
		}
	}
	if path == "" {
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("failed to read secret from file; falling back to direct env",
			"field", envTag, "path", path, "err", err)
		return
	}
	field.SetString(strings.TrimSpace(string(content)))
}

func trimQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') ||
		(s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// ─── Minimal .env file loader ────────────────────────────────────────────────
//
// Format:
//   # comments are ignored
//   KEY=value
//   KEY="quoted value with spaces"
//   KEY='single-quoted value'
//   export KEY=value          (the leading "export " is stripped)
//
// Only sets variables that aren't already in the environment, so real env
// vars override the file. This matches standard .env semantics and means
// systemd's EnvironmentFile= or shell exports take precedence if set.

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return fmt.Errorf("malformed line %d: %q", lineNo, line)
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		// Strip inline comments only when the value isn't quoted.
		if !isQuoted(value) {
			if hash := strings.Index(value, " #"); hash >= 0 {
				value = strings.TrimSpace(value[:hash])
			}
		}
		value = trimQuotes(value)

		if _, exists := os.LookupEnv(key); exists {
			continue // real env wins
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("setenv %s: %w", key, err)
		}
	}
	return scanner.Err()
}

func isQuoted(s string) bool {
	if len(s) < 2 {
		return false
	}
	return (s[0] == '"' && s[len(s)-1] == '"') ||
		(s[0] == '\'' && s[len(s)-1] == '\'')
}
