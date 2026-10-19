package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/debugreport"
	"github.com/ZachCurry13/cartrak/internal/store"
)

var lockWords = []string{"lock", "door", "hood", "trunk", "tailgate", "liftgate", "alarm"}
var neverWords = []string{"vin", "token", "email", "password", "latitude", "longitude", "position", "location", "gps"}

// allFields lists every field of one raw reading as sorted "name = value" lines, leaving out positions and anything
// that identifies the car or its owner. It lets the owner (and whoever helps) see everything the car reports.
func allFields(extra json.RawMessage) []string {
	f := map[string]any{}
	if json.Unmarshal(extra, &f) != nil {
		return nil
	}
	var out []string
	for k, v := range f {
		if containsAny(strings.ToLower(k), neverWords) {
			continue
		}
		val := strings.ReplaceAll(strings.TrimSpace(toString(v)), "\n", " ")
		if len(val) > 60 {
			val = val[:60]
		}
		out = append(out, k+" = "+val)
	}
	sort.Strings(out)
	if len(out) > 800 {
		out = out[:800]
	}
	return out
}

// lockFields picks the lock and door fields out of one raw reading, as sorted "name = value" lines.
func lockFields(extra json.RawMessage) []string {
	f := map[string]any{}
	if json.Unmarshal(extra, &f) != nil {
		return nil
	}
	var out []string
	for k, v := range f {
		lk := strings.ToLower(k)
		if !containsAny(lk, lockWords) || containsAny(lk, neverWords) {
			continue
		}
		val := strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toString(v)), "\n", " "))
		if len(val) > 80 {
			val = val[:80]
		}
		out = append(out, k+" = "+val)
	}
	sort.Strings(out)
	return out
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// debugReport is the text an owner can copy and send when something looks wrong: the last few hours of readings,
// the raw lock fields around each lock change, the trips and charging sessions, and the alerts sent. It leaves out
// positions, sign-ins, keys and the VIN. Admin only. ?hours=24 (1 to 72).
func (s *Server) debugReport(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours < 1 || hours > 72 {
		hours = 24
	}
	now := time.Now()
	from := now.Add(-time.Duration(hours) * time.Hour)
	rows, err := s.S.DebugRows(from.Unix(), now.Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	locks := map[int64][]string{}
	var prev *bool
	for i, row := range rows {
		changed := i == 0 || i == len(rows)-1 || (row.Locked == nil) != (prev == nil) || (row.Locked != nil && prev != nil && *row.Locked != *prev)
		prev = row.Locked
		if !changed || len(locks) > 80 {
			continue
		}
		if x, err := s.S.SampleAt(row.TS); err == nil && x != nil {
			locks[row.TS] = lockFields(x.Extra)
		}
	}
	var fields []string
	if x, err := s.S.LatestSample(); err == nil && x != nil {
		fields = allFields(x.Extra)
	}
	trips, _ := s.S.ListTrips(store.TripFilter{From: from.Unix(), Limit: 200})
	charges, _ := s.S.ListCharges(200)
	var recent []*store.Charge
	for _, c := range charges {
		if c.Open || c.EndTS == nil || *c.EndTS >= from.Unix() {
			recent = append(recent, c)
		}
	}
	alerts, _ := s.S.RecentAlerts(200)
	st := s.P.Status()
	var lastOK time.Time
	if st.LastOK > 0 {
		lastOK = time.Unix(st.LastOK, 0)
	}
	text := debugreport.Build(debugreport.Input{Version: s.Version, Now: now, Loc: s.defaultLoc(), From: from, Vehicle: s.vehicleInfo().Label,
		Linked: st.Linked, LastOK: lastOK, LastErr: st.LastError, Rows: rows, Locks: locks, Fields: fields, Trips: trips, Charges: recent, Alerts: alerts})
	ok(w, map[string]any{"text": text, "hours": hours, "readings": len(rows)})
}
