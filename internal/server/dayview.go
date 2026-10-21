package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type dayRoute struct {
	TripID int64        `json:"tripId"`
	Kind   string       `json:"kind"` // phone (recorded), estimate (a routing service's guess) or car (the car's own positions)
	Points [][2]float64 `json:"points"`
}

const (
	dayRoutePoints = 400
	dayEstimateMax = 8 // routing-service lookups one view may make for trips that have no estimate yet
	dayStopMinutes = 10
)

// routeForDay is the best line for one trip: a phone GPS track, else an estimated road, else the car's own positions.
func (s *Server) routeForDay(t *store.Trip, lookups *int) dayRoute {
	end := time.Now().Unix()
	if t.EndTS != nil {
		end = *t.EndTS
	}
	out := dayRoute{TripID: t.ID, Points: [][2]float64{}}
	var line [][2]float64
	for _, p := range s.phoneRoute(t.StartTS, end) {
		line = append(line, [2]float64{p.Lat, p.Lon})
	}
	if len(line) >= 2 {
		out.Kind, out.Points = "phone", thinLine(line, dayRoutePoints)
		return out
	}
	fetch := *lookups < dayEstimateMax
	if est := s.estimateFor(t, false); est != nil {
		out.Kind, out.Points = "estimate", thinLine(est.Points, dayRoutePoints)
		return out
	} else if fetch && s.routeLookupOn() {
		*lookups++
		if est := s.estimateFor(t, true); est != nil {
			out.Kind, out.Points = "estimate", thinLine(est.Points, dayRoutePoints)
			return out
		}
	}
	line = line[:0]
	ss, _ := s.S.TrackBetween(t.StartTS, end)
	for _, x := range ss {
		if x.Lat != nil && x.Lon != nil {
			p := [2]float64{*x.Lat, *x.Lon}
			if len(line) == 0 || line[len(line)-1] != p { // the car repeats its last position: keep each place once
				line = append(line, p)
			}
		}
	}
	out.Kind, out.Points = "car", thinLine(line, dayRoutePoints)
	return out
}

// dayView is everything on one day: the trips with their routes, the stops between them, and the day's totals.
// GET /api/day?date=2026-10-18 (today when empty, in the car's time zone)
func (s *Server) dayView(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	day := time.Now().In(loc)
	if d := r.URL.Query().Get("date"); d != "" {
		t, err := time.ParseInLocation("2006-01-02", d, loc)
		if err != nil {
			fail(w, 400, "the date must look like 2026-10-18")
			return
		}
		day = t
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 1)
	list, err := s.S.ListTrips(store.TripFilter{From: from.Unix(), To: to.Unix(), Limit: 200})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	sessions, err := s.chargeSessions(1000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var charges []history.Charge
	var costSum, kwhAdded float64
	var sum daySummary
	for _, c := range sessions {
		c := c
		if c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
			continue
		}
		charges = append(charges, history.Charge{ID: c.ID, Start: c.StartTS, End: c.EndTS, StartSoC: c.StartSoC, EndSoC: c.EndSoC,
			Lat: c.Lat, Lon: c.Lon, AddedKwh: c.AddedKwh, Cost: c.Cost, Where: c.Where, CostActual: c.CostActual, WallKwh: c.WallKwh})
		if !topUpBeforeDrive(c, list) {
			sum.addCharge(&c)
		}
		if c.AddedKwh != nil {
			kwhAdded += *c.AddedKwh
		}
		if c.Cost != nil {
			costSum += *c.Cost
		}
	}
	named, _ := s.S.ListNamedPlaces()
	now := time.Now().Unix()
	base := history.Build(list, charges, named, from.Unix(), to.Unix()-1, now)
	events := history.WithRemote(base, s.remoteFor(from.Unix(), to.Unix()-1), now) // the rows; the summary counts stops from base
	bases := s.baseFinder(s.cityNamer())
	s.placeEvents(events, list, named, bases)
	bases.cities.lookupLater()
	sort.SliceStable(events, func(i, j int) bool { return events[i].Start < events[j].Start }) // the day reads forwards
	capacity := s.capacityKwh()
	lookups := 0
	routes := []dayRoute{}
	var used, usedKm float64
	for i := len(list) - 1; i >= 0; i-- { // ListTrips is newest first
		t := list[i]
		if t.Open {
			continue
		}
		if rt := s.routeForDay(t, &lookups); len(rt.Points) >= 2 {
			routes = append(routes, rt)
		}
		if e, ok := trips.EnergyUsed(t, capacity); ok {
			used += e.Kwh
			usedKm += t.DistanceKm
		}
	}
	// Short stops are not stops: a few minutes parked between two drives is not worth a marker.
	keep := make([]history.Event, 0, len(events))
	dayEnd := to.Unix() - 1
	for _, e := range events {
		if e.End == nil && dayEnd < now { // a past day: the last stop ends with the day, not today
			end := dayEnd
			e.End = &end
		}
		if e.Kind == "parked" && e.End != nil && *e.End-e.Start < dayStopMinutes*60 {
			continue
		}
		keep = append(keep, e)
	}
	sum.finish(list, base, min(dayEnd, now))
	trip := s.dayCosts(&sum, from, used, usedKm, bases)
	ok(w, map[string]any{"date": from.Format("2006-01-02"), "summary": sum, "tz": loc.String(), "events": keep, "routes": routes, "currency": s.currency(),
		"totals": history.Sum(base, from.Unix(), to.Unix()-1, now),
		"energy": map[string]any{"usedKwh": used, "usedKm": usedKm, "addedKwh": kwhAdded, "cost": costSum}, "trip": trip})
}

// daySummary is the day in numbers for a plain-English paragraph: how long was driven and stood still, how many stops,
// what was charged and what it cost.
type daySummary struct {
	Trips          int      `json:"trips"`
	AvgMovingKph   float64  `json:"avgMovingKph"`
	TopKph         float64  `json:"topKph"`
	FirstStart     int64    `json:"firstStart"`
	LastEnd        int64    `json:"lastEnd"`
	Stops          int      `json:"stops"`      // parked stretches of ten minutes or more
	StoppedSec     int64    `json:"stoppedSec"` // time parked, not counting time spent charging
	LongestStop    int64    `json:"longestStopSec"`
	LongestAt      string   `json:"longestStopAt"`
	Sessions       int      `json:"sessions"`
	FastSessions   int      `json:"fastSessions"`
	ChargeSec      int64    `json:"chargeSec"`
	PeakKw         float64  `json:"peakKw"`
	AvgKw          float64  `json:"avgKw"`
	Receipts       int      `json:"receipts"`  // sessions whose cost is what was paid
	Estimated      int      `json:"estimated"` // sessions whose cost is worked out from a price
	Unpriced       int      `json:"unpriced"`
	CostPerKm      float64  `json:"costPerKm"`
	StartSoC       *float64 `json:"startSoc"`
	EndSoC         *float64 `json:"endSoc"`
	addedKwh, cost float64
}
