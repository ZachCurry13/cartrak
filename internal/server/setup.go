package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/trips"
)

// The setup checklist: suggested next steps, none of them required. What the server can tell is done comes from here;
// installing the app and taking the tour are known only on the phone. Each person hides the Home card for themselves.

func setupHiddenKey(uid int64) string  { return fmt.Sprintf("setup_hidden:%d", uid) }
func setupSkippedKey(uid int64) string { return fmt.Sprintf("setup_skipped:%d", uid) }

// skippedOf is the suggestions a person skipped, by key.
func (s *Server) skippedOf(uid int64) []string {
	out := []string{}
	for _, k := range strings.Split(s.core().GetKV(setupSkippedKey(uid)), ",") {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

func (s *Server) setupChecklist(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	out := map[string]any{"hidden": s.core().GetKV(setupHiddenKey(u.ID)) == "1", "skipped": s.skippedOf(u.ID)}
	alerts := 0
	if subs, err := s.core().ListPushSubs(u.ID); err == nil {
		alerts = len(subs)
	}
	out["alerts"] = alerts > 0
	if u.Role != "admin" {
		ok(w, out)
		return
	}
	named, _ := s.S.ListNamedPlaces()
	home, placePrice := false, false
	for _, p := range named {
		home = home || strings.EqualFold(p.Name, "home") || p.ChargeHome
		placePrice = placePrice || p.ChargePrice != nil
	}
	phones, _ := s.S.ListDevices()
	maintenance, _ := s.S.ListMaintenance()
	pr := s.prices()
	out["home"] = home
	out["price"] = pr.Home != nil || pr.Public != nil
	out["gas"] = s.Cfg != nil && s.Cfg.Gas().Set()
	out["placePrice"] = placePrice
	out["phone"] = len(phones) > 0
	out["maintenance"] = len(maintenance) > 0
	out["backup"] = s.S.GetKV(kvBackupTaken) != ""
	out["redetect"] = s.redetectOffer()
	ok(w, out)
}

// hideSetup takes the checklist card off this person's Home (hide) or puts it back (show); the checklist stays under More.
func (s *Server) hideSetup(w http.ResponseWriter, r *http.Request) {
	v := "1"
	if r.URL.Query().Get("show") == "1" {
		v = ""
	}
	_ = s.core().SetKV(setupHiddenKey(userFrom(r).ID), v)
	ok(w, map[string]bool{"hidden": v == "1"})
}

const kvDetectionSeen = "detection_seen"

// redetectOffer says whether to offer Fix a day: only after an update that finds trips or charging differently. A new
// installation, or one updating to the first version that knows about this, starts as seen.
func (s *Server) redetectOffer() bool {
	now := strconv.Itoa(trips.DetectionVersion)
	switch s.S.GetKV(kvDetectionSeen) {
	case now:
		return false
	case "":
		_ = s.S.SetKV(kvDetectionSeen, now)
		return false
	}
	return true
}

// redetectSeen puts the offer away until the next such update. POST /api/repair/seen
func (s *Server) redetectSeen(w http.ResponseWriter, r *http.Request) {
	_ = s.S.SetKV(kvDetectionSeen, strconv.Itoa(trips.DetectionVersion))
	ok(w, "seen")
}

// skipSetup skips one suggestion for this person, or brings it back with undo=1. POST /api/checklist/skip?key=gas
func (s *Server) skipSetup(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if len(key) == 0 || len(key) > 20 || strings.Trim(key, "abcdefghijklmnopqrstuvwxyz") != "" {
		fail(w, 400, "unknown suggestion")
		return
	}
	uid := userFrom(r).ID
	var keep []string
	for _, k := range s.skippedOf(uid) {
		if k != key {
			keep = append(keep, k)
		}
	}
	if r.URL.Query().Get("undo") != "1" {
		keep = append(keep, key)
	}
	_ = s.core().SetKV(setupSkippedKey(uid), strings.Join(keep, ","))
	ok(w, map[string]any{"skipped": keep})
}
