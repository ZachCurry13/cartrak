package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/drain"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/vhealth"
)

// vehicleHealth is the data behind the Vehicle health tab: the latest reading arranged by what
// matters for this type of car, plus a few daily trends. It never includes the VIN or location.
func (s *Server) vehicleHealth(w http.ResponseWriter, r *http.Request) {
	x, err := s.S.LatestSample()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	info := s.vehicleInfo()
	if x == nil {
		ok(w, map[string]any{"vehicle": info, "ready": false})
		return
	}
	f := map[string]any{}
	_ = json.Unmarshal(x.Extra, &f)
	now := time.Now()
	trends := vhealth.Trends{}
	for name, days := range map[string]int{"battery12v": 90, "capacityKwh": 365, "oilLifePct": 365} {
		daily := s.S.DailySeries
		if name == "capacityKwh" {
			daily = s.S.DailyMedian // the car's own estimate wanders, so show the middle reading of each day
		}
		if d, err := daily(name, now.AddDate(0, 0, -days).Unix()); err == nil {
			trends[name] = d
		}
	}
	pt := ford.Powertrain(info.Powertrain)
	report := vhealth.Build(f, pt, trends, vhealth.Options{Tested12VUntil: s.tested12V()})
	if pt == ford.EV || pt == ford.PHEV {
		if line, err := s.S.Timeline(now.AddDate(0, 0, -31).Unix(), now.Unix(), 0); err == nil {
			report.Sections = append(report.Sections, drain.Section(drain.Analyze(line, now.Unix()), s.capacityKwh(), s.defaultLoc()))
		}
	}
	ok(w, map[string]any{
		"vehicle":        info,
		"ready":          true,
		"asOf":           x.TS,
		"model":          modelName(f),
		"report":         report,
		"tested12vUntil": unixOrZero(s.tested12V()),
	})
}

// modelName is for example "2024 Mustang Mach-E", from Ford's garage record.
func modelName(f map[string]any) string {
	var parts []string
	for _, k := range []string{"garage.modelYear", "garage.modelName"} {
		if v := ford.Text(f, k); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

// tested12V is when the owner's "I had the 12 V battery tested and it is fine" pause ends (zero if none).
func (s *Server) tested12V() time.Time {
	if n, err := strconv.ParseInt(s.S.GetKV(store.KV12VTested), 10, 64); err == nil && n > 0 {
		return time.Unix(n, 0)
	}
	return time.Time{}
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// testedBattery records or clears the pause. POST {"days":90} pauses the 12 V warnings and alert for that
// long (default 90, at most 365); DELETE resumes them. Admin only.
func (s *Server) testedBattery(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		_ = s.S.SetKV(store.KV12VTested, "")
		ok(w, map[string]any{"until": 0})
		return
	}
	var in struct {
		Days int `json:"days"`
	}
	_ = decode(r, &in)
	if in.Days <= 0 {
		in.Days = 90
	}
	if in.Days > 365 {
		in.Days = 365
	}
	until := time.Now().AddDate(0, 0, in.Days).Unix()
	_ = s.S.SetKV(store.KV12VTested, strconv.FormatInt(until, 10))
	ok(w, map[string]any{"until": until})
}
