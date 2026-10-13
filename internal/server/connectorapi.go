package server

import (
	"context"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/connector"
)

// connectorInfo is the Cloudflare connector's version and update state. Admin only.
func (s *Server) connectorInfo(w http.ResponseWriter, r *http.Request) {
	if s.Conn == nil {
		ok(w, connector.Info{})
		return
	}
	ok(w, s.Conn.Info())
}

// connectorCheck asks GitHub now whether a newer connector exists.
func (s *Server) connectorCheck(w http.ResponseWriter, r *http.Request) {
	if s.Conn == nil {
		fail(w, 404, "the connector is not managed here")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := s.Conn.Check(ctx); err != nil {
		fail(w, http.StatusBadGateway, "could not reach GitHub: "+err.Error())
		return
	}
	ok(w, s.Conn.Info())
}

// connectorUpdate installs the newest connector in the background. Poll /api/connector for progress.
func (s *Server) connectorUpdate(w http.ResponseWriter, r *http.Request) {
	if s.Conn == nil || !s.Conn.Info().Supported {
		fail(w, 400, "installing a connector from the app works only on the Linux Docker image")
		return
	}
	if st := s.Conn.Info().Update.State; st == "working" {
		fail(w, http.StatusConflict, "an update is already running")
		return
	}
	go func() { _ = s.Conn.Update(context.Background()) }()
	w.WriteHeader(http.StatusAccepted)
	ok(w, "started")
}
