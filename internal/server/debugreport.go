package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/debugreport"
	"github.com/ZachCurry13/cartrak/internal/ford"
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

// lockFields sums up the locks, doors, hood and alarm of one raw reading in a few lines, with the time Ford says the car reported
// them. The raw fields are still in the full field list at the latest reading; this keeps a long report readable.
func lockFields(extra json.RawMessage) []string {
	f := map[string]any{}
	if json.Unmarshal(extra, &f) != nil {
		return nil
	}
	label := func(it map[string]any, key string) string {
		where := strings.ReplaceAll(strings.ToLower(fmt.Sprint(it[key])), "unspecified_", "")
		if side := strings.ToLower(fmt.Sprint(it["vehicleside"])); side != "" && side != "<nil>" && side != "unknown" {
			where = side + " " + where
		}
		return strings.ReplaceAll(where, "_", " ")
	}
	var out []string
	if at := ford.Updated(f, "doorStatus"); at != "" {
		out = append(out, "car reported at "+at)
	} else if at := ford.Updated(f, "doorLockStatus"); at != "" {
		out = append(out, "car reported at "+at)
	}
	var locks []string
	for _, it := range ford.ListItems(f, "doorLockStatus") {
		locks = append(locks, label(it, "vehicledoor")+" "+strings.ToLower(fmt.Sprint(it["value"])))
	}
	if len(locks) > 0 {
		out = append(out, "locks: "+strings.Join(locks, ", "))
	}
	var open []string
	doors := ford.ListItems(f, "doorStatus")
	for _, it := range doors {
		if v := strings.ToUpper(fmt.Sprint(it["value"])); v != "CLOSED" && v != "UNKNOWN" && v != "" {
			open = append(open, label(it, "vehicledoor")+" "+strings.ToLower(v))
		}
	}
	switch {
	case len(open) > 0:
		out = append(out, "doors: "+strings.Join(open, ", ")+" (the rest closed)")
	case len(doors) > 0:
		out = append(out, fmt.Sprintf("doors: all %d closed", len(doors)))
	}
	var rest []string
	for _, k := range []string{"hoodStatus", "alarmStatus", "panicAlarmStatus"} {
		if v := ford.Text(f, k); v != "" {
			rest = append(rest, strings.TrimSuffix(k, "Status")+" "+strings.ToLower(v))
		}
	}
	if len(rest) > 0 {
		out = append(out, strings.Join(rest, ", "))
	}
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
// positions, sign-ins, keys and the VIN. Admin only. ?hours=24 (1 to 336, two weeks).
func (s *Server) debugReport(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours < 1 || hours > 336 {
		hours = 24
	}
	if r.URL.Query().Get("all") == "1" { // every car's report in one, for an administrator of the whole installation
		if s.Cars == nil || !siteAdmin(r) {
			fail(w, 403, "only an administrator of the whole installation can make the report for every car")
			return
		}
		text, n := s.Cars.AllDebugReports(hours, metricReq(r))
		ok(w, map[string]any{"text": text, "hours": hours, "readings": n})
		return
	}
	text, n, err := s.DebugText(hours, metricReq(r))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"text": text, "hours": hours, "readings": n})
}

// DebugText is the debug report of this car for the last hours, and how many readings it covers.
func (s *Server) DebugText(hours int, metric bool) (string, int, error) {
	now := time.Now()
	from := now.Add(-time.Duration(hours) * time.Hour)
	rows, err := s.S.DebugRows(from.Unix(), now.Unix())
	if err != nil {
		return "", 0, err
	}
	s.withPackPower(rows, from.Unix(), now.Unix())
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
	points := map[int64][3]int{}
	for _, t := range trips {
		if t.EndTS == nil {
			continue
		}
		n, withPos, _ := s.S.SampleCounts(t.StartTS, *t.EndTS)
		phone, _ := s.S.PointsBetween(t.StartTS-routeSlack, *t.EndTS+routeSlack)
		points[t.ID] = [3]int{n, withPos, len(phone)}
	}
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
		MaxLines: min(6000, max(1100, hours*30)), Metric: metric, Linked: st.Linked, LastOK: lastOK, LastErr: st.LastError, Rows: rows, Locks: locks, Fields: fields, Trips: trips, Points: points, Charges: recent, Alerts: alerts,
		Polls: toPollInfo(s.P.RecentPolls(from)), Decisions: s.S.RecentDecisions(from.Unix()), Settings: s.debugSettings(), Curves: s.chargeCurves(recent)})
	return text, len(rows), nil
}
