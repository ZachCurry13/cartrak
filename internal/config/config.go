// Package config loads runtime settings from environment variables.
package config

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr     string
	DataDir  string
	Provider string // "fordpass" or "fordconnect"
	TZ       *time.Location

	FordClientID     string
	FordClientSecret string
	FordVIN          string
	FordUser         string // fordpass: account email
	FordPass         string // fordpass: account password
	FordRegion       string // fordpass: "usa" (default)
	FordRedirectURL  string // fordconnect: OAuth redirect URI
	FordAppID        string // fordconnect: Application-Id from the developer portal

	AdminUser string
	AdminPass string

	PollDriving time.Duration
	PollCharge  time.Duration
	PollParked  time.Duration
}

func Load() (*Config, error) {
	tz, err := time.LoadLocation(env("CARTRAK_TZ", "UTC"))
	if err != nil {
		tz = time.UTC
	}
	c := &Config{
		Addr:             env("CARTRAK_ADDR", ":8080"),
		DataDir:          env("CARTRAK_DATA", "./data"),
		Provider:         env("CARTRAK_PROVIDER", "fordpass"),
		TZ:               tz,
		FordClientID:     os.Getenv("FORD_CLIENT_ID"),
		FordClientSecret: os.Getenv("FORD_CLIENT_SECRET"),
		FordVIN:          os.Getenv("FORD_VIN"),
		FordUser:         os.Getenv("FORD_USERNAME"),
		FordPass:         os.Getenv("FORD_PASSWORD"),
		FordRegion:       env("FORD_REGION", "usa"),
		FordRedirectURL:  os.Getenv("FORD_REDIRECT_URL"),
		FordAppID:        os.Getenv("FORD_APPLICATION_ID"),
		AdminUser:        env("CARTRAK_ADMIN_USER", "admin"),
		AdminPass:        os.Getenv("CARTRAK_ADMIN_PASS"),
		PollDriving:      secs("CARTRAK_POLL_DRIVING_SEC", 30),
		PollCharge:       secs("CARTRAK_POLL_CHARGING_SEC", 120),
		PollParked:       secs("CARTRAK_POLL_PARKED_SEC", 600),
	}
	// Optional credentials file: {"clientId":"...","secret":"..."}
	if f := os.Getenv("CARTRAK_FORD_CREDS_FILE"); f != "" && c.FordClientID == "" {
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

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func secs(k string, def int) time.Duration {
	n, err := strconv.Atoi(os.Getenv(k))
	if err != nil || n <= 0 {
		n = def
	}
	return time.Duration(n) * time.Second
}
