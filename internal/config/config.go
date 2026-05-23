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
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Runtime configuration for the orchestrator daemon.
type Config struct {
	// ── VRRP topology ────────────────────────────────────────────────────────
	VRRPVirtualIP string `env:"VRRP_VIRTUAL_IP" default:""`

	// ── PostgreSQL ───────────────────────────────────────────────────────────
	PostgresDB         string `env:"POSTGRES_DB" default:"postgres"`
	PostgresUser       string `env:"POSTGRES_USER" default:"postgres"`
	PostgresPassword   string `env:"POSTGRES_PASSWORD" default:"" options:"file"`
	PostgresHost       string `env:"POSTGRES_HOST" default:"localhost"`
	PostgresPort       int    `env:"POSTGRES_PORT" default:"5432"`
	PostgresVolumeName string `env:"POSTGRES_VOLUME_NAME" default:"postgres.data"`

	// ── Valkey ───────────────────────────────────────────────────────────────
	ValkeyDB       int    `env:"VALKEY_DB" default:"0"`
	ValkeyHost     string `env:"VALKEY_HOST" default:"localhost"`
	ValkeyPort     int    `env:"VALKEY_PORT" default:"6379"`
	ValkeyPassword string `env:"VALKEY_PASSWORD" default:"" options:"file"`

	// ── WireGuard ────────────────────────────────────────────────────────────
	WireguardInterface string `env:"WIREGUARD_INTERFACE" default:"wg0"`

	// ── Files ──────────────────────────────────────────────────────────
	// File created by Keel, used to restrict orchestration decisions and actions.
	MaintenanceFlagPath string `env:"MAINTENANCE_FLAG_PATH" default:"/var/keel"`
	MaintenanceFlagFile string `env:"MAINTENANCE_FLAG_FILE" default:"maintenance_mode"`
	// File created by Postgres, used to keep Postgres from starting as primary.
	StandbySignalFile string `env:"STANDBY_SIGNAL_FILE" default:"standby.signal"`
	// File created by Keepalived, used by the notify_* scripts advertise the current local VRRP state.
	VRRPRolePath string `env:"VRRP_ROLE_PATH" default:"/run/vrrp"`
	VRRPRoleFile string `env:"VRRP_ROLE_FILE" default:"role"`
	// File created by Keel, used to advertise local state to peers and the local Keepalived daemon.
	StateFilePath string `env:"STATE_FILE_PATH" default:"/run/keel"`
	StateFile     string `env:"STATE_FILE" default:"state"`

	// ── Loop tuning ──────────────────────────────────────────────────────────
	TickInterval    time.Duration `env:"TICK_INTERVAL" default:"3s"`
	PeerDownStrikes int           `env:"PEER_DOWN_STRIKES" default:"3"`
	ProbeTimeout    time.Duration `env:"PROBE_TIMEOUT" default:"2s"`
	ActionTimeout   time.Duration `env:"ACTION_TIMEOUT" default:"60s"`
	APIPort         string        `env:"API_PORT" default:"9998"`

	// ── External tools ───────────────────────────────────────────────────────
	LoadBalancerIP string `env:"LOAD_BALANCER_IP" default:""`

	// ── Logging ──────────────────────────────────────────────────────────────
	LogLevel string `env:"LOG_LEVEL" default:"info" options:"toLower"`
	LogJSON  bool   `env:"LOG_JSON" default:"true"`
}

func Load(envPath string) (*Config, error) {
	if envPath != "" {
		if err := loadEnvFile(envPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	cfg := &Config{}
	loadFromEnv(cfg)
	applyOptions(cfg)

	// Validate required variables as expected by TestValidateRequiresPeerIPs
	// TODO Add required stuff for postgres and valkey
	if cfg.WireguardInterface == "" || cfg.VRRPVirtualIP == "" {
		return nil, fmt.Errorf("missing required configuration: WIREGUARD_INTERFACE and VRRP_VIRTUAL_IP must be specified")
	}

	return cfg, nil
}

func loadEnvFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		val = stripTrailingComment(val)
		val = trimQuotes(val)

		// Real environment variables take precedence over the .env file
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	return nil
}

func stripTrailingComment(val string) string {
	inQuote := false
	var quoteChar rune
	for i, r := range val {
		if r == '\'' || r == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			}
		}
		if r == '#' && !inQuote {
			return strings.TrimSpace(val[:i])
		}
	}
	return val
}

// loadFromEnv uses reflection to load configuration from environment variables.
func loadFromEnv(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	visitConfigFields(v, func(field reflect.Value, fieldType reflect.StructField) {
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			return
		}

		defaultValue := fieldType.Tag.Get("default")

		// Get the environment value directly first
		envValue := trimQuotes(os.Getenv(envTag))
		if envValue == "" {
			envValue = defaultValue
		}

		setFieldValueInternal(field, fieldType, envValue)
	})
}

// applyOptions processes special options for Config fields after initial load.
func applyOptions(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	visitConfigFields(v, func(field reflect.Value, fieldType reflect.StructField) {
		optionsTag := fieldType.Tag.Get("options")
		if optionsTag == "" {
			return
		}

		options := strings.SplitSeq(optionsTag, ",")
		for option := range options {
			switch strings.TrimSpace(option) {
			case "file":
				resolveFileBasedEnvVariable(field, fieldType)
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
	})
}

func visitConfigFields(v reflect.Value, fn func(reflect.Value, reflect.StructField)) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if fieldType.Anonymous {
			if field.Kind() == reflect.Struct {
				visitConfigFields(field, fn)
				continue
			}
			if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					if field.CanSet() {
						field.Set(reflect.New(field.Type().Elem()))
					} else {
						continue
					}
				}
				visitConfigFields(field.Elem(), fn)
				continue
			}
		}

		fn(field, fieldType)
	}
}

// resolveFileBasedEnvVariable checks if an environment variable with the suffix "_FILE" is set,
// reads the content of the file specified by that variable, and sets the corresponding field's value.
func resolveFileBasedEnvVariable(field reflect.Value, fieldType reflect.StructField) {
	// Only process string and []byte fields
	isString := field.Kind() == reflect.String
	isByteSlice := field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8
	if !isString && !isByteSlice {
		return
	}

	// Only process fields with the "env" tag
	envTag := fieldType.Tag.Get("env")
	if envTag == "" {
		return
	}

	// Check both double underscore (__FILE) and single underscore (_FILE) variants
	// Double underscore takes precedence
	var filePath string
	for _, suffix := range []string{"__FILE", "_FILE"} {
		if fp := os.Getenv(envTag + suffix); fp != "" {
			filePath = fp
			break
		}
	}

	if filePath == "" {
		return
	}

	fileContent, err := os.ReadFile(filePath) //nolint:gosec // file path intentionally comes from *_FILE env vars for Docker secrets
	if err != nil {
		slog.Warn("Failed to read secret from file, falling back to direct env var",
			"error", err)
		return
	}

	// Log when file value overrides a direct env var
	if os.Getenv(envTag) != "" {
		slog.Debug("Using secret from file, overriding direct env var")
	}

	if isString {
		field.SetString(strings.TrimSpace(string(fileContent)))
	} else {
		field.SetBytes(fileContent)
	}
}

// setFieldValueInternal sets a reflect.Value from a string based on the field's type.
func setFieldValueInternal(field reflect.Value, fieldType reflect.StructField, value string) {
	if !field.CanSet() {
		return
	}

	if field.Kind() == reflect.String {
		field.SetString(value)
		return
	}

	if field.Kind() == reflect.Bool {
		if b, err := strconv.ParseBool(value); err == nil {
			field.SetBool(b)
		}
		return
	}

	if field.Kind() == reflect.Uint32 {
		// Handle os.FileMode (which is uint32)
		if i, err := strconv.ParseUint(value, 8, 32); err == nil {
			field.SetUint(i)
		}
		return
	}

	if field.Kind() == reflect.Int {
		if i, err := strconv.Atoi(value); err == nil {
			field.SetInt(int64(i))
		}
		return
	}

	if field.Type() == reflect.TypeFor[time.Duration]() {
		applyDurationDefault := func(reason string) {
			envTag := fieldType.Tag.Get("env")
			defaultValue := fieldType.Tag.Get("default")

			if fallback, fallbackErr := time.ParseDuration(defaultValue); fallbackErr == nil {
				slog.Warn("Invalid duration for config field, using tagged default", //nolint:gosec // logging invalid config input for diagnostics is intentional here.
					"reason", reason,
					"field", envTag,
					"value", value,
					"default", defaultValue)
				field.SetInt(int64(fallback))
			} else {
				slog.Warn("Invalid duration for config field and invalid tagged default", //nolint:gosec // logging invalid config input for diagnostics is intentional here.
					"reason", reason,
					"field", envTag,
					"value", value,
					"default", defaultValue)
			}
		}

		if d, err := time.ParseDuration(value); err == nil {
			if d > 0 {
				field.SetInt(int64(d))
			} else {
				applyDurationDefault("Non-positive duration for config field")
			}
		} else {
			applyDurationDefault("Invalid duration for config field")
		}
		return
	}

	// Handle custom types based on underlying kind
	if field.Type().ConvertibleTo(reflect.TypeFor[string]()) {
		// String-based types like AppEnvironment
		field.Set(reflect.ValueOf(value).Convert(field.Type()))
	} else if field.Type() == reflect.TypeFor[os.FileMode]() {
		// os.FileMode
		if i, err := strconv.ParseUint(value, 8, 32); err == nil {
			field.Set(reflect.ValueOf(os.FileMode(i)))
		}
	}
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Postgres Connection String Constructor
func (c *Config) PostgresAddress() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:   c.PostgresHost + ":" + strconv.Itoa(c.PostgresPort),
		Path:   c.PostgresDB,
	}
	return u.String()
}

// Valkey Address String Constructor
func (c *Config) ValkeyAddress() string {
	u := url.URL{
		Host: c.ValkeyHost + ":" + strconv.Itoa(c.ValkeyPort),
	}
	return u.String()
}

// ListenAddr returns the effective address for the HTTP server to bind to.
// It uses LISTEN as the host (if set) and PORT for the port.
// func (c *Config) ListenAddr() string {
// 	host := strings.TrimSpace(c.Listen)
// 	port := c.Port
// 	if port == "" {
// 		port = "3552"
// 	}
// 	if host == "" {
// 		return ":" + port
// 	}
// 	return net.JoinHostPort(host, port)
// }

// MaskSensitive returns a copy of the config with sensitive fields masked.
// Useful for logging configuration without exposing secrets.
func (c *Config) MaskSensitive() map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			envTag = fieldType.Name
		}

		// Fields with "file" option are considered sensitive
		optionsTag := fieldType.Tag.Get("options")
		isSensitive := strings.Contains(optionsTag, "file")

		if isSensitive {
			// Mask sensitive values
			strVal := fmt.Sprintf("%v", field.Interface())
			if len(strVal) > 0 {
				result[envTag] = "****"
			} else {
				result[envTag] = "(empty)"
			}
		} else {
			result[envTag] = field.Interface()
		}
	}

	return result
}
