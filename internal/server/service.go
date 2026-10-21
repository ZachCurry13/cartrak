package server

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Service visits: when the car stays long enough at a place marked as a dealership, tire shop or repair shop, CarTrak asks after
// it leaves whether it was serviced. Every few minutes it looks at the last few days of trips for such a stop.

var serviceKindName = map[string]string{"dealer": "Dealership", "tire": "Tire shop", "repair": "Repair shop"}

func (s *Server) serviceLoop(ctx context.Context) {
	for {
		s.findServiceVisits(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
		}
	}
}

// servicePlaceAt is the service place a point is at, or nil.
func servicePlaceAt(lat, lon *float64, named []*store.NamedPlace) *store.NamedPlace {
	if lat == nil || lon == nil {
		return nil
	}
	for _, p := range named {
		if p.ServiceKind != "" && charging.DistanceM(charging.Point{Lat: p.Lat, Lon: p.Lon}, charging.Point{Lat: *lat, Lon: *lon}) <= p.RadiusM {
			return p
		}
	}
	return nil
}

// findServiceVisits records a pending visit for each finished stop at a service place that lasted at least the chosen time,
// and asks about it once.
func (s *Server) findServiceVisits(ctx context.Context, now time.Time) {
	minutes := 20
	if s.Cfg != nil {
		minutes = s.Cfg.ServiceMinutes()
	}
	if minutes <= 0 {
		return
	}
	named, _ := s.S.ListNamedPlaces()
	any := false
	for _, p := range named {
		any = any || p.ServiceKind != ""
	}
	if !any {
		return
	}
	ts, err := s.S.ListTrips(store.TripFilter{From: now.AddDate(0, 0, -3).Unix(), All: true})
	if err != nil {
		return
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].StartTS < ts[j].StartTS })
	for i := 0; i+1 < len(ts); i++ {
		t, next := ts[i], ts[i+1]
		if t.Open || t.EndTS == nil {
			continue
		}
		p := servicePlaceAt(t.EndLat, t.EndLon, named)
		stay := next.StartTS - *t.EndTS
		if p == nil || stay < int64(minutes)*60 || s.S.HasVisitAt(p.ID, *t.EndTS) {
			continue
		}
		v := &store.ServiceVisit{PlaceID: p.ID, PlaceName: p.Name, Kind: p.ServiceKind, Start: *t.EndTS, End: next.StartTS, OdoKm: t.EndOdo, Status: "pending"}
		if s.S.SaveVisit(v) != nil {
			continue
		}
		s.tell(ctx, push.Alert{Kind: push.ServiceVisit, Title: fmt.Sprintf("Was the car serviced at %s?", p.Name),
			Body: fmt.Sprintf("It was there %s. Tap to log what was done and what it cost.", stayText(stay)), URL: "#/service/" + strconv.FormatInt(v.ID, 10)})
	}
}

func stayText(sec int64) string {
	m := sec / 60
	if m < 60 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %d min", m/60, m%60)
}
