package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeest"
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
	known map[string][3]float64 // real road km, minutes at your pace and kWh per km, by the pair of places
	tried map[string]bool
	asked int

	paceKph  float64                   // how much faster than the road's own speed you drive on highway stretches
	energyAt func(kph float64) float64 // kWh per km at an average speed, from how this car has driven
}

func newRoadLegs(ctx context.Context, s *Server) *roadLegs {
	return &roadLegs{s: s, ctx: ctx, on: s.routeLookupOn(), known: map[string][3]float64{}, tried: map[string]bool{}}
}

func legKey(a, b tripchain.Station) string {
	return fmt.Sprintf("%.4f,%.4f>%.4f,%.4f", a.Lat, a.Lon, b.Lat, b.Lon)
}

func (r *roadLegs) real(a, b tripchain.Station) (km, min, kwhPerKm float64, ok bool) {
	v, ok := r.known[legKey(a, b)]
	return v[0], v[1], v[2], ok
}

// atPace is the time and the energy per km of a drive when the highway stretches are driven paceKph faster than the routing service
// drives them. On the rest of the road its own speed stands. The energy comes from this car's own efficiency at each speed.
func (r *roadLegs) atPace(p routeest.Pace) (min, kwhPerKm float64) {
	min = p.Minutes
	hwKph, hwAdjKph := 0.0, 0.0
	if p.HighwayMin > 0 {
		hwKph = p.HighwayKm / p.HighwayMin * 60
		hwAdjKph = hwKph + r.paceKph
		min = p.Minutes - p.HighwayMin + p.HighwayKm/hwAdjKph*60
	}
	if r.energyAt == nil || p.Km <= 0 {
		return min, 0
	}
	otherKm := p.Km - p.HighwayKm
	var kwh float64
	if p.HighwayKm > 0 {
		kwh += p.HighwayKm * r.energyAt(hwAdjKph)
	}
	if otherKm > 0 {
		otherMin := min - p.HighwayKm/max(hwAdjKph, 1)*60
		kph := 50.0
		if otherMin > 0 {
			kph = otherKm / otherMin * 60
		}
		kwh += otherKm * r.energyAt(kph)
	}
	return min, kwh / p.Km
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
			if p, err := r.s.routeClient().DrivePace(r.ctx, []routeest.Point{{a.Lat, a.Lon}, {b.Lat, b.Lon}}); err == nil && p.Km > 0 && p.Minutes > 0 {
				min, e := r.atPace(p)
				r.known[k] = [3]float64{p.Km, min, e}
				time.Sleep(250 * time.Millisecond) // the free routing service asks for a gentle pace
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
