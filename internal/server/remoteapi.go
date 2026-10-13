package server

import (
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/tunnel"
)

const (
	keyToken   = "CARTRAK_TUNNEL_TOKEN"
	keyAddress = "CARTRAK_PUBLIC_ADDRESS"
	keyEnabled = "CARTRAK_REMOTE_ENABLED"
)

type remoteView struct {
	Enabled   bool          `json:"enabled"`
	Address   string        `json:"address"`
	TokenSet  bool          `json:"tokenSet"`
	Locked    remoteLocks   `json:"locked"`
	Status    tunnel.Status `json:"status"`
	Installed bool          `json:"installed"`
	Log       []string      `json:"log"`
	URL       string        `json:"url"`
	Redirect  string        `json:"fordRedirect"`
	Port      string        `json:"port"`
	// ConnectorUpdate is true when a newer Cloudflare connector can be installed.
	ConnectorUpdate bool `json:"connectorUpdate"`
}

type remoteLocks struct {
	Token   bool `json:"token"`
	Address bool `json:"address"`
	Enabled bool `json:"enabled"`
}

// ApplyTunnel makes the connector match the saved settings. Call it at startup
// and after every change.
func (s *Server) ApplyTunnel() {
	if s.Tunnel == nil || s.Cfg == nil {
		return
	}
	s.Tunnel.Apply(s.Cfg.Value(keyEnabled) == "true", s.Cfg.Value(keyToken))
}

func (s *Server) remoteState() remoteView {
	v := remoteView{
		Enabled:  s.Cfg.Value(keyEnabled) == "true",
		Address:  s.Cfg.Value(keyAddress),
		TokenSet: s.Cfg.Value(keyToken) != "",
		Locked:   remoteLocks{Token: s.Cfg.Locked(keyToken), Address: s.Cfg.Locked(keyAddress), Enabled: s.Cfg.Locked(keyEnabled)},
		Port:     s.Port,
		Log:      []string{},
	}
	if v.Address != "" {
		v.URL = "https://" + v.Address
		v.Redirect = v.URL + "/ford/callback"
	}
	if s.Conn != nil {
		v.ConnectorUpdate = s.Conn.Info().Available
	}
	if s.Tunnel != nil {
		v.Status, v.Installed = s.Tunnel.Status(), s.Tunnel.Installed()
		v.Log = append(v.Log, s.Tunnel.Log()...)
	}
	return v
}

func (s *Server) getRemote(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil {
		fail(w, 503, "settings unavailable")
		return
	}
	ok(w, s.remoteState())
}

// saveRemote stores the tunnel token (write-only), the public address and the
// on/off switch, then starts or stops the connector.
func (s *Server) saveRemote(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil {
		fail(w, 503, "settings unavailable")
		return
	}
	var b struct {
		Token      *string `json:"token"`
		ClearToken bool    `json:"clearToken"`
		Address    *string `json:"address"`
		Enabled    *bool   `json:"enabled"`
	}
	if decode(r, &b) != nil {
		fail(w, 400, "bad request")
		return
	}
	var locked []string
	if b.Token != nil && strings.TrimSpace(*b.Token) != "" || b.ClearToken {
		if s.Cfg.Locked(keyToken) {
			locked = append(locked, "token")
		}
	}
	if b.Address != nil && s.Cfg.Locked(keyAddress) {
		locked = append(locked, "public address")
	}
	if b.Enabled != nil && s.Cfg.Locked(keyEnabled) {
		locked = append(locked, "on/off switch")
	}
	if len(locked) > 0 {
		fail(w, 409, "set by an environment variable or config.yaml, change it there: "+strings.Join(locked, ", "))
		return
	}

	newToken := s.Cfg.Value(keyToken)
	if b.ClearToken {
		newToken = ""
	}
	if b.Token != nil && strings.TrimSpace(*b.Token) != "" {
		t, err := tunnel.ExtractToken(*b.Token)
		if err != nil {
			fail(w, 400, "Tunnel token: "+err.Error())
			return
		}
		newToken = t
	}
	newAddr := s.Cfg.Value(keyAddress)
	if b.Address != nil {
		newAddr = ""
		if strings.TrimSpace(*b.Address) != "" {
			a, err := tunnel.NormalizeAddress(*b.Address)
			if err != nil {
				fail(w, 400, "Public address: "+err.Error())
				return
			}
			newAddr = a
		}
	}
	newOn := s.Cfg.Value(keyEnabled) == "true"
	if b.Enabled != nil {
		newOn = *b.Enabled
	}
	if newOn && newToken == "" {
		fail(w, 400, "Paste the tunnel token before turning remote access on.")
		return
	}

	if b.ClearToken || (b.Token != nil && strings.TrimSpace(*b.Token) != "") {
		_ = s.Cfg.Set(keyToken, newToken)
	}
	if b.Address != nil {
		_ = s.Cfg.Set(keyAddress, newAddr)
	}
	if b.Enabled != nil {
		v := "false"
		if newOn {
			v = "true"
		}
		_ = s.Cfg.Set(keyEnabled, v)
	}
	s.ApplyTunnel()
	ok(w, s.remoteState())
}
