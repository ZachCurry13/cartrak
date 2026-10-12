// Package settings resolves CarTrak's runtime settings from three layers.
// Highest first: environment variable, config.yaml, then the admin panel
// (stored in the database). A value set by env or config.yaml is "locked":
// the admin panel shows it but cannot change it.
package settings

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Def describes one setting that can be edited in the admin panel.
type Def struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Help        string `json:"help"`
	Placeholder string `json:"placeholder"`
	Secret      bool   `json:"secret"`
}

var Defs = []Def{
	{Key: "FORD_CLIENT_ID", Label: "Ford client ID", Help: "From your app in Ford's developer portal."},
	{Key: "FORD_CLIENT_SECRET", Label: "Ford client secret", Secret: true, Help: "Stored on the server only. It is never shown again or sent to the browser."},
	{Key: "FORD_APPLICATION_ID", Label: "Ford Application-Id (older FordConnect 1 only)", Help: "Not needed for FordConnect 2.0. Leave empty unless you use the older provider fordconnect1."},
	{Key: "FORD_REDIRECT_URL", Label: "Ford redirect URL", Placeholder: "https://cartrak.example.com/ford/callback",
		Help: "Must exactly match a redirect URI registered on your Ford app. Registered http://localhost? Enter exactly that: after signing in you paste the address Ford sends you to. Registered a CarTrak address? Use that, and linking finishes by itself."},
	{Key: "FORD_AUTH_URL", Label: "Ford sign-in address (advanced)", Placeholder: "leave empty to use the built-in address",
		Help: "Only change this if Ford's documentation gives a different sign-in (authorize) address for FordConnect 2.0."},
	{Key: "FORD_VIN", Label: "Vehicle ID (optional)", Help: "Only needed if several vehicles are authorized. Default: the first one."},
	{Key: "CARTRAK_TZ", Label: "Time zone", Placeholder: "America/Chicago", Help: "IANA name. Work-hour rules use this time zone."},
}

// Lookup reads a process-level (env or config.yaml) value and its source.
type Lookup func(key string) (value, source string)

type KV interface {
	GetKV(k string) string
	SetKV(k, v string) error
}

type Settings struct {
	KV   KV
	Proc Lookup
}

func New(kv KV, proc Lookup) *Settings { return &Settings{KV: kv, Proc: proc} }

func dbKey(k string) string { return "setting:" + k }

// Get returns the effective value and its source: env, config.yaml, admin or default.
func (s *Settings) Get(key string) (string, string) {
	if v, src := s.Proc(key); v != "" {
		return v, src
	}
	if v := s.KV.GetKV(dbKey(key)); v != "" {
		return v, "admin"
	}
	switch key {
	case "FORD_REDIRECT_URL":
		if v := s.KV.GetKV("derived_redirect"); v != "" {
			return v, "default"
		}
	case "CARTRAK_TZ":
		return "UTC", "default"
	}
	return "", "default"
}

func (s *Settings) Value(key string) string { v, _ := s.Get(key); return v }

// Locked reports whether env or config.yaml controls the key.
func (s *Settings) Locked(key string) bool { v, _ := s.Proc(key); return v != "" }

// Set stores an admin-panel value; an empty value clears it.
func (s *Settings) Set(key, val string) error { return s.KV.SetKV(dbKey(key), strings.TrimSpace(val)) }

// Location is the configured time zone, UTC if it cannot be loaded.
func (s *Settings) Location() *time.Location {
	if l, err := time.LoadLocation(s.Value("CARTRAK_TZ")); err == nil {
		return l
	}
	return time.UTC
}

// Configured is true when the Ford client ID and secret needed to link are present.
func (s *Settings) Configured() bool {
	return s.Value("FORD_CLIENT_ID") != "" && s.Value("FORD_CLIENT_SECRET") != ""
}

// Valid checks a value before it is saved.
func Valid(key, val string) error {
	val = strings.TrimSpace(val)
	if len(val) > 2000 {
		return fmt.Errorf("too long")
	}
	if v, ok := validators[key]; ok {
		if err := v(val); err != nil {
			return err
		}
	}
	switch key {
	case "CARTRAK_TZ":
		if val != "" {
			if _, err := time.LoadLocation(val); err != nil {
				return fmt.Errorf("unknown time zone %q (use a name like America/Chicago)", val)
			}
		}
	case "FORD_REDIRECT_URL":
		if val != "" {
			u, err := url.Parse(val)
			if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && (strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1")))) {
				return fmt.Errorf("must be a full https:// address, like https://cartrak.example.com/ford/callback")
			}
		}
	}
	return nil
}

// Known reports whether key is an editable setting.
func Known(key string) (Def, bool) {
	for _, d := range Defs {
		if d.Key == key {
			return d, true
		}
	}
	return Def{}, false
}

func init() {
	// FORD_AUTH_URL must be a full https address, like the redirect URL.
	validators["FORD_AUTH_URL"] = func(val string) error {
		if val == "" {
			return nil
		}
		u, err := url.Parse(val)
		if err != nil || u.Host == "" || u.Scheme != "https" {
			return fmt.Errorf("must be a full https:// address")
		}
		return nil
	}
}

var validators = map[string]func(string) error{}
