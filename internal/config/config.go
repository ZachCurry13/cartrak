// Package config loads settings. Precedence, highest first:
// environment variable, then config.yaml in the config folder, then default.
// config.yaml uses the same names as the environment variables.
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed config.example.yaml
var Example string

type Config struct {
	Addr      string
	DataDir   string
	ConfigDir string
	Provider  string // "fordconnect" or "demo"
	TZ        *time.Location

	FordClientID     string
	FordClientSecret string
	FordAppID        string // Application-Id from the developer portal
	FordRedirectURL  string // OAuth redirect URI registered with Ford
	FordVIN          string // optional vehicle id; default is the first authorized one

	AdminUser   string
	AdminPass   string
	ResetAdmin  bool // CARTRAK_RESET_ADMIN=1: set the admin password from CARTRAK_ADMIN_PASS on start
	TrustProxy  bool
	SessionDays int

	src source

	PollDriving time.Duration
	PollCharge  time.Duration
	PollParked  time.Duration
}

type source struct{ file map[string]any }

// get returns the environment value, else the config.yaml value, else "".
// lookup returns the value and where it came from: "env", "config.yaml" or "".
func (s source) lookup(k string) (string, string) {
	if v := os.Getenv(k); v != "" {
		return v, "env"
	}
	if v, ok := s.file[k]; ok && v != nil && fmt.Sprint(v) != "" {
		return fmt.Sprint(v), "config.yaml"
	}
	return "", ""
}

func (s source) get(k string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	if v, ok := s.file[k]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func (s source) str(k, def string) string {
	if v := s.get(k); v != "" {
		return v
	}
	return def
}

func (s source) secs(k string, def int) time.Duration {
	n, err := strconv.Atoi(s.get(k))
	if err != nil || n <= 0 {
		n = def
	}
	return time.Duration(n) * time.Second
}

// Load reads the config folder (CARTRAK_CONFIG, default ./config).
func Load() (*Config, error) {
	dir := firstNonEmpty(os.Getenv("CARTRAK_CONFIG"), "./config")
	src := source{file: map[string]any{}}
	if b, err := os.ReadFile(filepath.Join(dir, "config.yaml")); err == nil {
		if err := yaml.Unmarshal(b, &src.file); err != nil {
			return nil, fmt.Errorf("config.yaml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	// Always refresh the documented example next to the real file.
	if err := os.MkdirAll(dir, 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "config.example.yaml"), []byte(Example), 0o644)
	}

	tz, err := time.LoadLocation(src.str("CARTRAK_TZ", "UTC"))
	if err != nil {
		tz = time.UTC
	}
	c := &Config{
		Addr:             src.str("CARTRAK_ADDR", ":8080"),
		DataDir:          src.str("CARTRAK_DATA", "./data"),
		ConfigDir:        dir,
		Provider:         src.str("CARTRAK_PROVIDER", "fordconnect"),
		TZ:               tz,
		FordClientID:     src.get("FORD_CLIENT_ID"),
		FordClientSecret: src.get("FORD_CLIENT_SECRET"),
		FordAppID:        src.get("FORD_APPLICATION_ID"),
		FordRedirectURL:  src.get("FORD_REDIRECT_URL"),
		FordVIN:          src.get("FORD_VIN"),
		AdminUser:        src.str("CARTRAK_ADMIN_USER", "admin"),
		AdminPass:        src.get("CARTRAK_ADMIN_PASS"),
		ResetAdmin:       src.get("CARTRAK_RESET_ADMIN") == "1",
		TrustProxy:       src.get("CARTRAK_TRUST_PROXY") != "false",
		SessionDays:      int(src.secs("CARTRAK_SESSION_DAYS", 30) / time.Second),
		src:              src,
		PollDriving:      src.secs("CARTRAK_POLL_DRIVING_SEC", 30),
		PollCharge:       src.secs("CARTRAK_POLL_CHARGING_SEC", 120),
		PollParked:       src.secs("CARTRAK_POLL_PARKED_SEC", 600),
	}
	// Optional JSON credentials file: {"clientId":"...","secret":"..."}
	if f := src.get("CARTRAK_FORD_CREDS_FILE"); f != "" && c.FordClientID == "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var v struct{ ClientID, Secret string }
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		c.FordClientID, c.FordClientSecret = v.ClientID, v.Secret
	}
	return c, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Lookup returns a process-level setting and its source ("env" or "config.yaml").
// Empty means it is not set there.
func (c *Config) Lookup(key string) (string, string) { return c.src.lookup(key) }
