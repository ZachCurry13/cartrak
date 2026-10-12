package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"poller": s.P.Status(), "canLink": s.Link != nil, "configured": s.Cfg == nil || s.Cfg.Configured()})
}

func (s *Server) latest(w http.ResponseWriter, r *http.Request) {
	x, err := s.S.LatestSample()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, x) // null before the first reading
}

func tripFilter(r *http.Request) store.TripFilter {
	q := r.URL.Query()
	f := store.TripFilter{Category: q.Get("category")}
	f.From, _ = strconv.ParseInt(q.Get("from"), 10, 64)
	f.To, _ = strconv.ParseInt(q.Get("to"), 10, 64)
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	if f.Category != "work" && f.Category != "personal" {
		f.Category = ""
	}
	return f
}

func (s *Server) listTrips(w http.ResponseWriter, r *http.Request) {
	ts, err := s.S.ListTrips(tripFilter(r))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, ts)
}

func (s *Server) tripDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil {
		fail(w, 404, "no such trip")
		return
	}
	end := time.Now().Unix()
	if t.EndTS != nil {
		end = *t.EndTS
	}
	ss, _ := s.S.SamplesBetween(t.StartTS, end)
	ok(w, map[string]any{"trip": t, "samples": ss})
}

func (s *Server) patchTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var b struct{ Category, Notes string }
	if decode(r, &b) != nil || (b.Category != "work" && b.Category != "personal") {
		fail(w, 400, "category must be work or personal")
		return
	}
	if err := s.S.SetTripCategory(id, b.Category, b.Notes); err != nil {
		fail(w, 500, "db error")
		return
	}
	t, _ := s.S.GetTrip(id)
	ok(w, t)
}

func (s *Server) listCharges(w http.ResponseWriter, r *http.Request) {
	cs, err := s.S.ListCharges(200)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, cs)
}

// summary totals distance and trip counts per category for a date range.
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	f.Category, f.Limit = "", 1000
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	type agg struct {
		Trips      int     `json:"trips"`
		DistanceKm float64 `json:"distanceKm"`
		KWh        float64 `json:"kwh"`
	}
	out := map[string]*agg{"work": {}, "personal": {}}
	for _, t := range ts {
		a := out[t.Category]
		if a == nil || t.Open {
			continue
		}
		a.Trips++
		a.DistanceKm += t.DistanceKm
	}
	ok(w, out)
}

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	f.Limit = 1000
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="cartrak-trips.csv"`)
	c := csv.NewWriter(w)
	_ = c.Write([]string{"id", "start", "end", "category", "source", "distance_km", "start_soc", "end_soc", "max_speed_kph", "notes"})
	for _, t := range ts {
		end := ""
		if t.EndTS != nil {
			end = time.Unix(*t.EndTS, 0).UTC().Format(time.RFC3339)
		}
		_ = c.Write([]string{strconv.FormatInt(t.ID, 10), time.Unix(t.StartTS, 0).UTC().Format(time.RFC3339), end,
			t.Category, t.CategorySource, fmt.Sprintf("%.2f", t.DistanceKm), fl(t.StartSoC), fl(t.EndSoC),
			fmt.Sprintf("%.0f", t.MaxSpeedKph), csvSafe(t.Notes)})
	}
	c.Flush()
}

func fl(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *p)
}

// csvSafe stops spreadsheet formula injection from user-entered notes.
func csvSafe(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}
