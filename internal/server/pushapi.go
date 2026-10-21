package server

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const maxDevicesPerPerson = 10

// defaultAlerts are what a newly added device starts with; each can be changed.
var defaultAlerts = []string{push.ChargeComplete, push.LowTire, push.Low12V}

type subIn struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	Label  string   `json:"label"`
	Events []string `json:"events"`
	Quiet  string   `json:"quiet"`
}

// pushConfig gives the page what it needs to subscribe, and this person's devices.
func (s *Server) pushConfig(w http.ResponseWriter, r *http.Request) {
	if s.Push == nil {
		fail(w, 503, "notifications are not available")
		return
	}
	key, err := s.Push.PublicKey()
	if err != nil {
		fail(w, 500, "could not prepare notifications")
		return
	}
	subs, _ := s.core().ListPushSubs(userFrom(r).ID)
	ok(w, map[string]any{"publicKey": key, "kinds": push.Kinds, "devices": subs, "defaults": defaultAlerts})
}

// validate checks a device's choices and fills in defaults.
func (s *Server) validate(in *subIn) string {
	if !s.Push.AllowAny && !push.AllowedEndpoint(in.Endpoint) {
		return "that is not a known push service address"
	}
	if raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(in.Keys.P256dh, "=")); err != nil || len(raw) != 65 {
		return "the device's key is not valid"
	}
	if raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(in.Keys.Auth, "=")); err != nil || len(raw) < 8 {
		return "the device's secret is not valid"
	}
	if msg := checkPrefs(in.Events, in.Quiet); msg != "" {
		return msg
	}
	return ""
}

func checkPrefs(events []string, quiet string) string {
	for _, e := range events {
		if !push.ValidKind(e) {
			return "unknown alert " + e
		}
	}
	if !push.ValidQuiet(quiet) {
		return "quiet hours must look like 22:00-07:00"
	}
	return ""
}

func (s *Server) pushSubscribe(w http.ResponseWriter, r *http.Request) {
	var in subIn
	if s.Push == nil || decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := s.validate(&in); msg != "" {
		fail(w, 400, msg)
		return
	}
	u := userFrom(r)
	if subs, _ := s.core().ListPushSubs(u.ID); len(subs) >= maxDevicesPerPerson {
		if _, err := s.core().GetPushSub(u.ID, in.Endpoint); err != nil { // not just re-saving a known device
			fail(w, 400, "remove a device first (limit 10)")
			return
		}
	}
	label := strings.TrimSpace(in.Label)
	if label == "" || len(label) > 40 {
		label = "This device"
	}
	if len(in.Events) == 0 {
		in.Events = defaultAlerts
	}
	err := s.core().SavePushSub(&store.PushSub{UserID: u.ID, Endpoint: in.Endpoint, P256dh: in.Keys.P256dh, Auth: in.Keys.Auth, Label: label, Events: in.Events, Quiet: in.Quiet})
	if err != nil {
		fail(w, 500, "could not save the device")
		return
	}
	s.pushConfig(w, r)
}

func (s *Server) pushPrefs(w http.ResponseWriter, r *http.Request) {
	var in subIn
	if s.Push == nil || decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := checkPrefs(in.Events, in.Quiet); msg != "" {
		fail(w, 400, msg)
		return
	}
	if err := s.core().SetPushPrefs(userFrom(r).ID, in.Endpoint, in.Events, in.Quiet); errors.Is(err, store.ErrNoDevice) {
		fail(w, 404, "this device is not registered for alerts any more: turn alerts on here again")
		return
	} else if err != nil {
		fail(w, 500, "could not save")
		return
	}
	s.pushConfig(w, r)
}

func (s *Server) pushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var in subIn
	if s.Push == nil || decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	_ = s.core().DeletePushSub(userFrom(r).ID, in.Endpoint)
	s.pushConfig(w, r)
}

// pushTest sends a test alert to one of the person's own devices and says how it went.
func (s *Server) pushTest(w http.ResponseWriter, r *http.Request) {
	var in subIn
	if s.Push == nil || decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	sub, err := s.core().GetPushSub(userFrom(r).ID, in.Endpoint)
	if err != nil {
		fail(w, 404, "that device is not set up for alerts")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := s.Push.SendTo(ctx, sub, push.Alert{Kind: "test", Title: "CarTrak test", Body: "Alerts are working on this device.", URL: "#/notifications"}); err != nil {
		fail(w, 502, err.Error())
		return
	}
	ok(w, "sent")
}
