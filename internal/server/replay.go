package server

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// replayPoint is one moment of a trip played back: where, how fast, the battery and the distance so far.
type replayPoint struct {
	TS  int64    `json:"ts"`
	Lat float64  `json:"lat"`
	Lon float64  `json:"lon"`
	Kph *float64 `json:"kph,omitempty"`
	SoC *float64 `json:"soc,omitempty"`
	Km  float64  `json:"km"` // distance since the start, along the route, scaled to the trip's recorded distance
}

const maxReplayPoints = 600

// tripReplay is a finished trip as moments in time, for playing it back on its map. The phone's GPS track is used when
// there is one (finer), otherwise the car's own positions; the battery and distance come from the car's readings, matched by
// time. GET /api/trips/{id}/replay
func (s *Server) tripReplay(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	var pts []replayPoint
	source := "phone"
	for _, p := range s.phoneRoute(t.StartTS, *t.EndTS) {
		if p.TS >= t.StartTS-60 && p.TS <= *t.EndTS+60 {
			pts = append(pts, replayPoint{TS: p.TS, Lat: p.Lat, Lon: p.Lon, Kph: p.SpeedKph})
		}
	}
	if len(pts) < 2 {
		pts, source = pts[:0], "car"
		ss, _ := s.S.TrackBetween(t.StartTS, *t.EndTS)
		for _, x := range ss {
			if x.Lat != nil && x.Lon != nil {
				pts = append(pts, replayPoint{TS: x.TS, Lat: *x.Lat, Lon: *x.Lon, Kph: x.SpeedKph})
			}
		}
	}
	if source == "car" { // follow the road the trip's map shows, not straight lines between the car's few positions
		if est := s.estimateFor(t, true); est != nil {
			if road := onRoad(est.Points, pts, t.StartTS, *t.EndTS); len(road) >= 2 {
				pts, source = road, "estimate"
			}
		}
	}
	if len(pts) < 2 {
		fail(w, 409, "this trip has too few positions to play back")
		return
	}
	tl, _ := s.S.Timeline(t.StartTS-120, *t.EndTS+120, 2000)
	matchReadings(pts, tl)
	alongRoute(pts, t.DistanceKm)
	if len(pts) > maxReplayPoints {
		step := float64(len(pts)-1) / float64(maxReplayPoints-1)
		thinned := make([]replayPoint, 0, maxReplayPoints)
		for i := 0; i < maxReplayPoints; i++ {
			thinned = append(thinned, pts[int(float64(i)*step+0.5)])
		}
		pts = thinned
	}
	ok(w, map[string]any{"points": pts, "start": t.StartTS, "end": *t.EndTS, "km": t.DistanceKm, "tz": t.TZ, "source": source})
}

// matchReadings gives each point the battery level (and a speed, when it has none) from the car reading nearest in time.
func matchReadings(pts []replayPoint, tl []store.TimePoint) {
	if len(tl) == 0 {
		return
	}
	sort.Slice(tl, func(i, j int) bool { return tl[i].TS < tl[j].TS })
	j := 0
	for i := range pts {
		for j+1 < len(tl) && absI(tl[j+1].TS-pts[i].TS) <= absI(tl[j].TS-pts[i].TS) {
			j++
		}
		x := tl[j]
		if absI(x.TS-pts[i].TS) > 600 {
			continue
		}
		pts[i].SoC = x.SoC
		if pts[i].Kph == nil {
			pts[i].Kph = x.SpeedKph
		}
	}
}

// alongRoute fills in the distance so far: the length of the route up to each point, scaled so the end matches the trip.
func alongRoute(pts []replayPoint, tripKm float64) {
	var sum float64
	for i := 1; i < len(pts); i++ {
		sum += charging.DistanceM(charging.Point{Lat: pts[i-1].Lat, Lon: pts[i-1].Lon}, charging.Point{Lat: pts[i].Lat, Lon: pts[i].Lon}) / 1000
		pts[i].Km = sum
	}
	if sum > 0 && tripKm > 0 {
		for i := range pts {
			pts[i].Km *= tripKm / sum
		}
	}
}

func absI(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
