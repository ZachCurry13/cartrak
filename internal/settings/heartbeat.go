package settings

import (
	"fmt"
	"net/url"
	"strings"
)

// HeartbeatURLKey is the address of an outside uptime monitor that CarTrak pings every few minutes. CarTrak cannot announce its own
// outage; the monitor does, when the pings stop, and again when they come back.
const HeartbeatURLKey = "CARTRAK_HEARTBEAT_URL"

func init() {
	Defs = append(Defs, Def{Key: HeartbeatURLKey, Label: "Uptime monitor address", Secret: true, Placeholder: "https://hc-ping.com/…",
		Help: "Optional. An address from an uptime monitor (for example a free Healthchecks.io check) that CarTrak asks every five minutes. When the asking stops, the monitor tells you CarTrak is down, and tells you again when it is back. It stays on the server."})
	validators[HeartbeatURLKey] = func(v string) error {
		if strings.TrimSpace(v) == "" {
			return nil
		}
		u, err := url.Parse(strings.TrimSpace(v))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return fmt.Errorf("this must be an address starting with https://")
		}
		return nil
	}
}

// HeartbeatURL is the uptime monitor's address, "" when not set.
func (s *Settings) HeartbeatURL() string { return strings.TrimSpace(s.Value(HeartbeatURLKey)) }
