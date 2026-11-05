package server

import (
	"context"
	"fmt"

	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

const (
	verifyRounds  = 4  // times a plan is checked against real roads and made again
	verifyLookups = 30 // the most routing-service questions one plan may ask
)

// roadLegs checks a plan's legs against real roads. A plan is first made with straight lines made longer, which is wrong across water
// or round a lake (a leg that looks 130 miles can be 220 by road). Each leg of the plan is then asked of the routing service, and the plan
// is made again knowing the real distances, until every leg in it is a real road or the questions run out.
type roadLegs struct {
	s     *Server
	ctx   context.Context
	on    bool
	known map[string][2]float64 // real road km and minutes, by the pair of places
	tried map[string]bool
	asked int
}

func newRoadLegs(ctx context.Context, s *Server) *roadLegs {
	return &roadLegs{s: s, ctx: ctx, on: s.routeLookupOn(), known: map[string][2]float64{}, tried: map[string]bool{}}
}

func legKey(a, b tripchain.Station) string {
	return fmt.Sprintf("%.4f,%.4f>%.4f,%.4f", a.Lat, a.Lon, b.Lat, b.Lon)
}

func (r *roadLegs) real(a, b tripchain.Station) (km, min float64, ok bool) {
	v, ok := r.known[legKey(a, b)]
	return v[0], v[1], ok
}

// learn asks for the road between each leg of the plan not asked about yet, and says how many it asked.
func (r *roadLegs) learn(days []tripchain.Day) int {
	n := 0
	for _, d := range days {
		for i := 1; i < len(d.Stops); i++ {
			a, b := d.Stops[i-1].Station, d.Stops[i].Station
			k := legKey(a, b)
			if r.tried[k] || r.asked >= verifyLookups || r.ctx.Err() != nil {
				continue
			}
			r.tried[k] = true
			r.asked++
			n++
			if km, min, estimated := r.s.legRoad(r.ctx, true, stopPoint{Name: a.Name, Lat: a.Lat, Lon: a.Lon}, stopPoint{Name: b.Name, Lat: b.Lat, Lon: b.Lon}); !estimated {
				r.known[k] = [2]float64{km, min}
			}
		}
	}
	return n
}

// plan makes the plan, then checks it on real roads and makes it again until nothing new was learned.
func (r *roadLegs) plan(p tripchain.Params, stations []tripchain.Station, vias []tripchain.Via) ([]tripchain.Day, error) {
	p.Real = r.real
	days, err := tripchain.Plan(p, stations, vias)
	for round := 0; r.on && err == nil && round < verifyRounds; round++ {
		if r.learn(days) == 0 {
			break
		}
		days, err = tripchain.Plan(p, stations, vias)
	}
	return days, err
}

// exact says whether every leg of the plan is a real road, not an estimate.
func (r *roadLegs) exact(days []tripchain.Day) bool {
	for _, d := range days {
		for i := 1; i < len(d.Stops); i++ {
			if _, ok := r.known[legKey(d.Stops[i-1].Station, d.Stops[i].Station)]; !ok {
				return false
			}
		}
	}
	return len(days) > 0
}
