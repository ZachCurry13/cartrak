package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/mileage"
	"github.com/ZachCurry13/cartrak/internal/report"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// tripsOf is the trips that started in [start, end) and are in one of the wanted categories (all of them when none is named).
func (s *Server) tripsOf(start, end time.Time, want map[string]bool) ([]*store.Trip, error) {
	all, err := s.S.TripsBetween(start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	var picked []*store.Trip
	for _, t := range all {
		if len(want) == 0 || want[t.Category] {
			picked = append(picked, t)
		}
	}
	return picked, nil
}

// downloadMileageLog is the FedEx report: the mileage log, one row a day with the origin city, the cities the car was parked in,
// a description and the miles, as a spreadsheet. Work trips unless other categories are chosen.
func (s *Server) downloadMileageLog(w http.ResponseWriter, r *http.Request, start, end time.Time, want map[string]bool, loc *time.Location) {
	q := r.URL.Query()
	if len(want) == 0 && q.Get("categories") == "" { // "all" is a choice; only no choice at all means work trips
		want = map[string]bool{"work": true}
	}
	picked, err := s.tripsOf(start, end, want)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	rep := mileage.Build(ctx, picked, s.places(), mileage.Options{Description: strings.TrimSpace(q.Get("description")), Default: loc, RoundUp: q.Get("rounding") != "nearest"})
	name := fmt.Sprintf("cartrak-mileage-log-%s-%s", start.Format("2006-01-02"), end.AddDate(0, 0, -1).Format("2006-01-02"))
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasSuffix(r.URL.Path, ".csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
		_ = report.MileageCSV(w, rep)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.xlsx"`)
	_ = report.MileageWorkbook(rep).Write(w)
}

// lookupReportPlaces: POST /api/reports/lookup, with the same choices as a download. It names the cities where the chosen trips started
// and ended, a few at a time (the lookup service is slow on purpose), and says how many are left, so a report has its cities when it is
// made. Admin only, because it asks the lookup service.
func (s *Server) lookupReportPlaces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	loc := s.defaultLoc()
	start, end, err := reportPeriod(q.Get("weeks"), q.Get("year"), q.Get("from"), q.Get("to"), loc)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	want := map[string]bool{}
	for _, k := range strings.Split(q.Get("categories"), ",") {
		if k = strings.TrimSpace(k); k != "" && k != "all" {
			want[k] = true
		}
	}
	if q.Get("template") == "fedex" && len(want) == 0 && q.Get("categories") == "" {
		want["work"] = true
	}
	picked, err := s.tripsOf(start, end, want)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	pl := s.places()
	seen := map[string]bool{}
	var todo [][2]float64
	add := func(lat, lon *float64) {
		if lat == nil || lon == nil {
			return
		}
		k := geocode.Key(*lat, *lon)
		if seen[k] {
			return
		}
		seen[k] = true
		if _, ok := pl.Cached(*lat, *lon); !ok {
			todo = append(todo, [2]float64{*lat, *lon})
		}
	}
	for _, t := range picked {
		if !t.Open {
			add(t.StartLat, t.StartLon)
			add(t.EndLat, t.EndLon)
		}
	}
	const perCall = 12
	ctx := r.Context()
	looked, failed := 0, ""
	for _, p := range todo {
		if looked >= perCall || ctx.Err() != nil {
			break
		}
		if _, err := pl.Lookup(ctx, p[0], p[1]); err != nil {
			failed = err.Error()
			break
		}
		looked++
	}
	ok(w, map[string]any{"pending": len(todo) - looked, "looked": looked, "error": failed})
}
