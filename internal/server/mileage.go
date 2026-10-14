package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/mileage"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// offPlaces names only places already remembered; it never asks anyone.
type offPlaces struct{ s *store.Store }

func (o offPlaces) Cached(lat, lon float64) (string, bool) {
	return o.s.GetPlace(geocode.Key(lat, lon))
}
func (offPlaces) Lookup(context.Context, float64, float64) (string, error) {
	return "", fmt.Errorf("city lookup is switched off in Settings")
}

// geoState holds the shared lookup client, so its one-request-a-second spacing holds across reports.
type geoState struct {
	mu sync.Mutex
	c  *geocode.Client
}

func (s *Server) places() mileage.Places {
	if s.Cfg != nil && !s.Cfg.GeocodeEnabled() {
		return offPlaces{s.S}
	}
	s.geo.mu.Lock()
	defer s.geo.mu.Unlock()
	if s.geo.c == nil {
		s.geo.c = &geocode.Client{Cache: s.S, UserAgent: "CarTrak/" + s.Version + " (+https://github.com/ZachCurry13/cartrak)", MinGap: s.GeoGap}
	}
	if s.Cfg != nil {
		s.geo.c.SetBaseURL(s.Cfg.GeocodeURL())
	}
	return s.geo.c
}

// mileageReport builds the business mileage log for calendar weeks (weeks=26-28[&year=2026]) or
// dates (from=2026-09-15&to=2026-10-02). Admin only, because it may ask the city lookup service.
func (s *Server) mileageReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	loc := time.UTC
	if s.Cfg != nil {
		loc = s.Cfg.Location()
	}
	start, end, err := reportPeriod(q.Get("weeks"), q.Get("year"), q.Get("from"), q.Get("to"), loc)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	cat := q.Get("category")
	switch cat {
	case "", "work":
		cat = "work"
	case "personal":
	default:
		cat = "" // all trips
	}
	trips, err := s.S.ListTrips(store.TripFilter{From: start.Unix(), To: end.Unix(), Category: cat, Limit: 1000})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	rep := mileage.Build(ctx, trips, s.places(), mileage.Options{Description: q.Get("description"), Default: loc, RoundUp: q.Get("rounding") != "nearest"})
	var rate *float64 // a deduction only makes sense for work trips
	if cat == "work" {
		rate = s.mileageRate()
	}
	ok(w, map[string]any{
		"from": start.Format("2006-01-02"), "to": end.AddDate(0, 0, -1).Format("2006-01-02"),
		"report": rep, "tsv": rep.TSV(false), "tsvFull": rep.TSV(true),
		"lookupOn": s.Cfg == nil || s.Cfg.GeocodeEnabled(), "rate": rate, "currency": s.currency(),
	})
}

// reportPeriod reads either ISO calendar weeks or an inclusive date range, in the default time zone.
func reportPeriod(weeks, year, from, to string, loc *time.Location) (start, end time.Time, err error) {
	if weeks != "" {
		a, b, err := mileage.ParseWeeks(weeks)
		if err != nil {
			return start, end, err
		}
		y := time.Now().In(loc).Year()
		if year != "" {
			if y, err = strconv.Atoi(year); err != nil {
				return start, end, fmt.Errorf("the year must be a number like 2026")
			}
		}
		return mileage.WeekRange(y, a, b, loc)
	}
	if from == "" || to == "" {
		return start, end, fmt.Errorf("choose calendar weeks (for example 26-28) or a from and to date")
	}
	if start, err = time.ParseInLocation("2006-01-02", from, loc); err != nil {
		return start, end, fmt.Errorf("dates must look like 2026-09-15")
	}
	last, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil || last.Before(start) {
		return start, end, fmt.Errorf("the to date must be on or after the from date")
	}
	return start, last.AddDate(0, 0, 1), nil
}

// mileageRate is the deduction per mile the owner set, or nil.
func (s *Server) mileageRate() *float64 {
	if s.Cfg == nil {
		return nil
	}
	return s.Cfg.MileageRate()
}
