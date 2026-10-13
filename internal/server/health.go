package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/tunnel"
)

// healthItem is one plain-language check on the Status screen.
type healthItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	State  string `json:"state"`            // ok | warn | bad | info
	Href   string `json:"href,omitempty"`   // where to fix it (admins)
	Action string `json:"action,omitempty"` // button label for Href
}

func ago(sec int64) string {
	switch {
	case sec < 90:
		return "just now"
	case sec < 5400:
		return fmt.Sprintf("%d min ago", (sec+30)/60)
	case sec < 129600:
		return fmt.Sprintf("%d h ago", (sec+1800)/3600)
	}
	return fmt.Sprintf("%d days ago", sec/86400)
}

// health answers "is it working?": the Ford link, whether readings are arriving,
// remote access, and updates, each with what to do next. It never includes secrets.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	st := s.P.Status()
	var items []healthItem
	add := func(h healthItem) { items = append(items, h) }

	// 1. Ford account
	switch {
	case s.Link != nil && s.Cfg != nil && !s.Cfg.Configured():
		add(healthItem{"ford", "Ford account", "Your Ford client ID and secret are not entered yet.", "bad", "#/admin", "Enter Ford settings"})
	case s.Link != nil && !st.Linked:
		add(healthItem{"ford", "Ford account", "Not linked yet. Sign in at Ford and approve your car.", "bad", "#/connect", "Connect Ford"})
	case !st.Linked:
		add(healthItem{"ford", "Ford account", "Using the " + st.Provider + " data source.", "info", "", ""})
	default:
		add(healthItem{"ford", "Ford account", "Linked.", "ok", "", ""})
	}

	// 2. Readings
	latest, _ := s.S.LatestSample()
	switch {
	case !st.Linked:
		// nothing to say until linked
	case st.RateLimited:
		wait := st.NextTry - now
		if wait < 0 {
			wait = 0
		}
		add(healthItem{"readings", "Car readings", fmt.Sprintf("Ford is limiting how often CarTrak may ask (normal after linking or pressing Sync now). Waiting, next try in about %d s. Nothing to do.", wait), "warn", "", ""})
	case st.LastError != "":
		add(healthItem{"readings", "Car readings", "The last update failed: " + st.LastError, "warn", "#/ford", "Open Ford connection"})
	case latest == nil:
		add(healthItem{"readings", "Car readings", "Waiting for the first reading from Ford. This can take a few minutes after linking.", "warn", "", ""})
	default:
		age := now - latest.TS
		if age > 3*3600 {
			add(healthItem{"readings", "Car readings", "The last reading was " + ago(age) + ". The car may be asleep or out of coverage; readings resume when it reports.", "warn", "", ""})
		} else {
			add(healthItem{"readings", "Car readings", "Last reading " + ago(age) + ".", "ok", "", ""})
		}
	}

	// 3. Remote access
	if s.Tunnel != nil && s.Cfg != nil {
		on := s.Cfg.Value(keyEnabled) == "true"
		ts := s.Tunnel.Status()
		switch {
		case !on:
			add(healthItem{"remote", "Remote access", "Not set up. Optional: lets you use CarTrak away from home.", "info", "#/remote", "Set up remote access"})
		case ts.State == tunnel.Connected:
			add(healthItem{"remote", "Remote access", "Connected to Cloudflare" + addressNote(s.Cfg.Value(keyAddress)) + ".", "ok", "", ""})
		case ts.State == tunnel.NotInstalled:
			add(healthItem{"remote", "Remote access", ts.Message, "warn", "#/remote", "Open Remote access"})
		default:
			add(healthItem{"remote", "Remote access", "Switched on but not connected yet: " + ts.Message, "warn", "#/remote", "Open Remote access"})
		}
	}

	// 4. Updates
	if s.Upd != nil {
		if u := s.Upd.Info(); u.Available {
			add(healthItem{"update", "CarTrak version", fmt.Sprintf("Version %s is available (you have %s).", u.Latest, s.Version), "info", "#/about", "See what's new"})
		}
	}

	overall := "ok"
	for _, it := range items {
		if it.State == "bad" {
			overall = "bad"
			break
		}
		if it.State == "warn" {
			overall = "warn"
		}
	}
	ok(w, map[string]any{"overall": overall, "items": items})
}

func addressNote(a string) string {
	if a == "" {
		return ""
	}
	return " (" + a + ")"
}
