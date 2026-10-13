package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/mileage"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tzlookup"
)

// gapOut is a gap with where the car was at each end, taken from the car's own position readings, and the
// city names already remembered for those places (no lookup is made here).
type gapOut struct {
	mileage.Gap
	StartLat  *float64 `json:"startLat"`
	StartLon  *float64 `json:"startLon"`
	EndLat    *float64 `json:"endLat"`
	EndLon    *float64 `json:"endLon"`
	StartCity string   `json:"startCity,omitempty"`
	EndCity   string   `json:"endCity,omitempty"`
}

func (s *Server) defaultLoc() *time.Location {
	if s.Cfg != nil {
		return s.Cfg.Location()
	}
	return time.UTC
}

// mileageCheck compares the car's odometer with the recorded trips for the period, the same period
// choice as the mileage log (weeks=26-28 or from= and to=). Admin only.
func (s *Server) mileageCheck(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start, end, err := reportPeriod(q.Get("weeks"), q.Get("year"), q.Get("from"), q.Get("to"), s.defaultLoc())
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	pts, err := s.S.OdoBetween(start.Unix(), end.Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var trips []*store.Trip
	if len(pts) > 1 {
		lo, hi := pts[0].Km, pts[0].Km
		for _, p := range pts {
			lo, hi = min(lo, p.Km), max(hi, p.Km)
		}
		if trips, err = s.S.TripsOverlappingOdo(lo, hi); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	c := mileage.CheckTrips(pts, trips)
	places := s.places()
	gaps := make([]gapOut, 0, len(c.Gaps))
	for _, g := range c.Gaps {
		o := gapOut{Gap: g}
		if track, err := s.S.TrackBetween(g.FromTS-1800, g.ToTS+1800); err == nil {
			o.StartLat, o.StartLon = positionAt(track, g.FromTS, false)
			o.EndLat, o.EndLon = positionAt(track, g.ToTS, true)
		}
		if o.StartLat != nil {
			o.StartCity, _ = places.Cached(*o.StartLat, *o.StartLon)
		}
		if o.EndLat != nil {
			o.EndCity, _ = places.Cached(*o.EndLat, *o.EndLon)
		}
		gaps = append(gaps, o)
	}
	ok(w, map[string]any{
		"from": start.Format("2006-01-02"), "to": end.AddDate(0, 0, -1).Format("2006-01-02"),
		"check": map[string]any{"readings": c.Readings, "drivenKm": c.DrivenKm, "coveredKm": c.CoveredKm, "missingKm": c.MissingKm, "ok": c.Ok, "gaps": gaps},
	})
}

// positionAt is the car's position at the reading nearest to ts: at or before it, or at or after it.
func positionAt(track []*store.Sample, ts int64, after bool) (lat, lon *float64) {
	var pick *store.Sample
	for _, x := range track {
		if x.Lat == nil || x.Lon == nil {
			continue
		}
		if after && x.TS >= ts {
			return x.Lat, x.Lon
		}
		if !after && x.TS <= ts {
			pick = x
		}
	}
	if pick != nil {
		return pick.Lat, pick.Lon
	}
	return nil, nil
}

type manualIn struct {
	StartTS  int64    `json:"startTs"`
	EndTS    int64    `json:"endTs"`
	StartOdo float64  `json:"startOdo"`
	EndOdo   float64  `json:"endOdo"`
	StartLat *float64 `json:"startLat"`
	StartLon *float64 `json:"startLon"`
	EndLat   *float64 `json:"endLat"`
	EndLon   *float64 `json:"endLon"`
	Category string   `json:"category"`
}

// addMissedTrip records a trip the car drove that CarTrak did not catch, from a gap the check found.
func (s *Server) addMissedTrip(w http.ResponseWriter, r *http.Request) {
	var in manualIn
	if err := decode(r, &in); err != nil {
		fail(w, 400, "bad request")
		return
	}
	switch {
	case in.Category != "work" && in.Category != "personal":
		fail(w, 400, "choose work or personal")
		return
	case in.StartTS <= 0 || in.EndTS <= in.StartTS || in.EndTS-in.StartTS > 14*86400:
		fail(w, 400, "the times do not make a trip")
		return
	case in.EndOdo <= in.StartOdo || in.EndOdo-in.StartOdo > 2000:
		fail(w, 400, "the odometer readings do not make a trip")
		return
	}
	existing, err := s.S.TripsOverlappingOdo(in.StartOdo, in.EndOdo)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	pts := []store.OdoPoint{{TS: in.StartTS, Km: in.StartOdo}, {TS: in.EndTS, Km: in.EndOdo}}
	if len(mileage.CheckTrips(pts, existing).Gaps) == 0 {
		fail(w, http.StatusConflict, "a trip already covers those miles")
		return
	}
	loc := s.defaultLoc()
	if in.StartLat != nil && in.StartLon != nil {
		loc = tzlookup.Location(*in.StartLat, *in.StartLon, loc)
	}
	t := &store.Trip{StartTS: in.StartTS, EndTS: &in.EndTS, StartLat: in.StartLat, StartLon: in.StartLon, EndLat: in.EndLat, EndLon: in.EndLon,
		StartOdo: &in.StartOdo, EndOdo: &in.EndOdo, DistanceKm: in.EndOdo - in.StartOdo, Category: in.Category,
		Notes: "Added from the mileage check", TZ: loc.String()}
	if line, err := s.S.Timeline(in.StartTS, in.EndTS, 0); err == nil && len(line) > 0 {
		t.StartSoC, t.EndSoC = line[0].SoC, line[len(line)-1].SoC
	}
	if err := s.S.AddManualTrip(t); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"id": t.ID})
}

// extendTrip adds the miles the car drove just before a trip was noticed, or just after it was closed.
// POST /api/trips/{id}/extend {"edge":"start"|"end","odo":12345.6}
func (s *Server) extendTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Edge string  `json:"edge"`
		Odo  float64 `json:"odo"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "bad request")
		return
	}
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil || t.StartOdo == nil || t.EndOdo == nil {
		fail(w, 404, "no such trip")
		return
	}
	a, b := *t.StartOdo, *t.EndOdo
	switch {
	case in.Edge == "start" && in.Odo < a && a-in.Odo <= 50:
		a = in.Odo
	case in.Edge == "end" && in.Odo > b && in.Odo-b <= 50:
		b = in.Odo
	default:
		fail(w, 400, fmt.Sprintf("that does not extend the %s of the trip", in.Edge))
		return
	}
	if err := s.S.SetTripOdometer(id, a, b, b-a); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"distanceKm": b - a})
}
