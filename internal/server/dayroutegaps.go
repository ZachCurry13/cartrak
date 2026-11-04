package server

import (
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// phoneGapM is how far the phone's track may stop short of where the trip started or ended before the gap is drawn.
const phoneGapM = 800.0

// fillPhoneGaps returns the phone track with the parts of the trip it missed drawn in as dotted estimates: when the phone's GPS
// was switched on partway through a drive, the track starts (or ends) away from where the trip did, and a line alone would look
// like a shorter trip. The gap follows the car's own positions when it has them, and is a straight line otherwise.
func (s *Server) fillPhoneGaps(t *store.Trip, end int64, phone []store.Point, track dayRoute) []dayRoute {
	out := []dayRoute{}
	first, last := phone[0], phone[len(phone)-1]
	if t.StartLat != nil && t.StartLon != nil && trips.Haversine(*t.StartLat, *t.StartLon, first.Lat, first.Lon) > phoneGapM {
		out = append(out, s.gapLine(t.ID, [2]float64{*t.StartLat, *t.StartLon}, t.StartTS, first.TS, [2]float64{first.Lat, first.Lon}))
	}
	out = append(out, track)
	if t.EndLat != nil && t.EndLon != nil && t.EndTS != nil && trips.Haversine(*t.EndLat, *t.EndLon, last.Lat, last.Lon) > phoneGapM {
		out = append(out, s.gapLine(t.ID, [2]float64{last.Lat, last.Lon}, last.TS, end, [2]float64{*t.EndLat, *t.EndLon}))
	}
	return out
}

// gapLine joins two points through the car's positions between two times.
func (s *Server) gapLine(id int64, a [2]float64, from, to int64, b [2]float64) dayRoute {
	line := [][2]float64{a}
	if ss, err := s.S.TrackBetween(from, to); err == nil {
		for _, x := range ss {
			if x.Lat != nil && x.Lon != nil {
				if p := [2]float64{*x.Lat, *x.Lon}; p != line[len(line)-1] {
					line = append(line, p)
				}
			}
		}
	}
	line = append(line, b)
	return dayRoute{TripID: id, Kind: "estimate", Points: thinLine(line, dayRoutePoints)}
}
