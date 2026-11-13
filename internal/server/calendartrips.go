package server

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A trip that ends at an appointment's place, from an hour and a half before it starts until it ends, went to that appointment.
const (
	apptNearM  = 400.0
	apptEarly  = 90 * time.Minute
	apptLookTo = 36 * time.Hour // trips that ended this recently are looked at
)

// calendarLoop names the trips that went to appointments, every ten minutes, and sends the weather warnings and the evening
// heads-up when they are switched on.
func (s *Server) calendarLoop(ctx context.Context) {
	for {
		s.nameTripsFromCalendar(ctx, time.Now())
		s.weatherWatch(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Minute):
		}
	}
}

// nameTripsFromCalendar gives a finished trip with no notes the title of the appointment it went to.
func (s *Server) nameTripsFromCalendar(ctx context.Context, now time.Time) int {
	if s.Cfg == nil || s.Cfg.CalendarURL() == "" {
		return 0
	}
	ts, err := s.S.ListTrips(store.TripFilter{From: now.Add(-apptLookTo).Unix(), Limit: 200})
	if err != nil {
		return 0
	}
	byDay := map[string][]appointment{}
	named := 0
	for _, t := range ts {
		if t.Open || t.EndTS == nil || t.EndLat == nil || t.EndLon == nil || strings.TrimSpace(t.Notes) != "" {
			continue
		}
		end := time.Unix(*t.EndTS, 0)
		key := end.In(s.Cfg.Location()).Format("2006-01-02")
		list, seen := byDay[key]
		if !seen {
			list, _, _ = s.appointments(ctx, end)
			byDay[key] = list
		}
		a := wentTo(list, end, *t.EndLat, *t.EndLon)
		if a == nil {
			continue
		}
		if s.Cfg.CalendarWork() && t.CategorySource != "manual" {
			err = s.S.SetTripCategory(t.ID, "work", a.Title) // counted as work by the owner's choice, kept like one made by hand
		} else {
			err = s.S.SetTripNotes(t.ID, a.Title)
		}
		if err != nil {
			log.Printf("calendar: naming trip %d failed: %v", t.ID, err)
			continue
		}
		s.S.LogDecision("trip_calendar", "trip "+strconv.FormatInt(t.ID, 10)+" named after the appointment it went to")
		named++
	}
	return named
}

func wentTo(list []appointment, end time.Time, lat, lon float64) *appointment {
	for i := range list {
		a := &list[i]
		if a.Place == nil || a.AllDay || end.Before(time.Unix(a.Start, 0).Add(-apptEarly)) || end.After(time.Unix(a.End, 0)) {
			continue
		}
		if charging.DistanceM(charging.Point{Lat: lat, Lon: lon}, charging.Point{Lat: a.Place.Lat, Lon: a.Place.Lon}) <= apptNearM {
			return a
		}
	}
	return nil
}
