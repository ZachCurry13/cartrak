package server

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/stops"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Stops are the parked stretches between trips, such as time spent in a customer's parking lot. See package stops.

// stopsFor lists the stops that began in [from, to), oldest first, with their place names filled in.
func (s *Server) stopsFor(from, to int64) ([]stops.Stop, error) {
	pad := int64(stops.MaxSec) + 86400 // a stop's trip began before it, and the next trip after it
	ts, err := s.S.ListTrips(store.TripFilter{From: from - pad, To: to + pad, All: true})
	if err != nil {
		return nil, err
	}
	in := make([]stops.Trip, 0, len(ts))
	for _, t := range ts {
		end := int64(0)
		if t.EndTS != nil && !t.Open {
			end = *t.EndTS
		}
		in = append(in, stops.Trip{ID: t.ID, StartTS: t.StartTS, EndTS: end, EndLat: t.EndLat, EndLon: t.EndLon, Category: t.Category})
	}
	labels, err := s.S.StopLabels()
	if err != nil {
		return nil, err
	}
	sl := map[int64]stops.Label{}
	for id, l := range labels {
		sl[id] = stops.Label{Category: l.Category, Note: l.Note}
	}
	charges, _ := s.S.ListCharges(2000)
	home := s.namedHome()
	skip := func(lat, lon *float64, start, end int64) bool {
		if home != nil && lat != nil && lon != nil && charging.DistanceM(charging.Point{Lat: *lat, Lon: *lon}, *home) <= 150 {
			return true
		}
		for _, c := range charges {
			if c.StartTS >= start-300 && c.StartTS <= end { // a charge began during the stop: that time is charging, not working
				return true
			}
		}
		return false
	}
	all := stops.Build(in, sl, skip, time.Now().Unix())
	named, _ := s.S.ListNamedPlaces()
	geo := s.places()
	var out []stops.Stop
	for _, st := range all {
		if st.Start < from || st.Start >= to {
			continue
		}
		st.Place = places.Name(st.Lat, st.Lon, named)
		if st.Place == "" && st.Lat != nil && st.Lon != nil {
			st.Place, _ = geo.Cached(*st.Lat, *st.Lon)
		}
		out = append(out, st)
	}
	return out, nil
}

// stopsRange reads ?from=YYYY-MM-DD&to=YYYY-MM-DD (the last 30 days by default) as the instants [from, to) in the car's time zone.
func (s *Server) stopsRange(r *http.Request) (from, to int64) {
	loc := s.defaultLoc()
	today := time.Now().In(loc)
	day := func(v string, def time.Time) time.Time {
		if t, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
			return t
		}
		return time.Date(def.Year(), def.Month(), def.Day(), 0, 0, 0, 0, loc)
	}
	f := day(r.URL.Query().Get("from"), today.AddDate(0, 0, -29))
	t := day(r.URL.Query().Get("to"), today).AddDate(0, 0, 1)
	return f.Unix(), t.Unix()
}

// stopsList: GET /api/stops. The stops, newest first, and the time each day and the whole range came to.
func (s *Server) stopsList(w http.ResponseWriter, r *http.Request) {
	from, to := s.stopsRange(r)
	list, err := s.stopsFor(from, to)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	loc := s.defaultLoc()
	days := stops.Totals(list, func(ts int64) string { return time.Unix(ts, 0).In(loc).Format("2006-01-02") })
	var all, work int
	for _, d := range days {
		all += d.Minutes
		work += d.Work
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	if list == nil {
		list = []stops.Stop{}
	}
	ok(w, map[string]any{"stops": list, "days": days, "minutes": all, "workMinutes": work, "tz": loc.String()})
}

// stopsLabel: PUT /api/stops/{id}. Sets a stop's category by hand and a note such as a customer's name. An empty category goes back to
// the one the trip that arrived there had.
func (s *Server) stopsLabel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Category string `json:"category"`
		Note     string `json:"note"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	if _, err := s.S.GetTrip(id); err != nil {
		fail(w, 404, "no such stop")
		return
	}
	if len(in.Note) > 200 || (in.Category != "" && !s.S.CategoryExists(in.Category)) {
		fail(w, 400, "choose one of your trip categories and keep the note under 200 characters")
		return
	}
	if err := s.S.SetStopLabel(id, store.StopLabel{Category: in.Category, Note: in.Note}); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]bool{"ok": true})
}

// safeCell keeps a spreadsheet from reading a note as a formula.
func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// stopsCSV: GET /api/stops.csv. The stops of the range for a spreadsheet.
func (s *Server) stopsCSV(w http.ResponseWriter, r *http.Request) {
	from, to := s.stopsRange(r)
	list, err := s.stopsFor(from, to)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	loc := s.defaultLoc()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="stops.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Date", "Start", "End", "Minutes", "Place", "Category", "Note"})
	for _, st := range list {
		a, b := time.Unix(st.Start, 0).In(loc), time.Unix(st.End, 0).In(loc)
		_ = cw.Write([]string{a.Format("2006-01-02"), a.Format("15:04"), b.Format("15:04"), strconv.Itoa(st.Minutes), safeCell(st.Place), st.Category, safeCell(st.Note)})
	}
	cw.Flush()
}
