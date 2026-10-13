package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const maxDevices = 20

// listDevices shows the paired phones and the latest things they reported (admin only).
func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	ds, err := s.S.ListDevices()
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	es, _ := s.S.RecentEvents(15)
	ok(w, map[string]any{"devices": ds, "events": es})
}

// addDevice creates a device and returns its token once; only a hash is kept.
func (s *Server) addDevice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "bad request")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 40 {
		fail(w, 400, "give the device a name of 1 to 40 characters")
		return
	}
	if ds, _ := s.S.ListDevices(); len(ds) >= maxDevices {
		fail(w, 400, "remove an unused device first (limit "+strconv.Itoa(maxDevices)+")")
		return
	}
	token := "ct_" + randToken()
	id, err := s.S.CreateDevice(name, store.TokenHash(token))
	if err != nil {
		fail(w, 500, "could not create the device")
		return
	}
	ok(w, map[string]any{"id": id, "name": name, "token": token})
}

func (s *Server) removeDevice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "bad id")
		return
	}
	if err := s.S.DeleteDevice(id); err != nil {
		fail(w, 500, err.Error())
		return
	}
	ok(w, "removed")
}
