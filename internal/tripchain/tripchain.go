// Package tripchain picks the Superchargers for a long trip: the fastest set of stops that keeps every leg inside the driving
// time, distance and battery you allow, and splits the trip into days of at most so many hours. Distances here are estimates (a
// straight line made longer): the chosen stops are then planned on real roads and weather by the route planner.
package tripchain

import (
	"errors"
	"math"
	"sort"
)

// Station is a charger the trip may stop at.
type Station struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Kw   int     `json:"kw"`
	PS   int     `json:"ps,omitempty"`  // the site's PlugShare number, for its reviews
	Net  string  `json:"net,omitempty"` // the network when it is not Tesla
	// Penalty is minutes added to using this charger, so a preferred network wins unless the other saves more than that.
	Penalty float64 `json:"-"`
}

// Params are the trip and the limits.
type Params struct {
	Start, Dest Station
	StartPct    float64 // the charge on leaving
	ReservePct  float64
	TargetPct   float64 // what a stop charges to
	CapacityKwh float64
	KwhPerKm    float64 // the car's usual use at highway speed
	CarKw       float64 // the most power the car takes
	MaxLegKm    float64 // 0 for no limit
	MaxLegMin   float64 // 0 for no limit
	DayMin      float64 // the longest day on the road, charging included; 0 for no limit
	SpeedKph    float64 // average speed used to turn distance into time
}

// Stop is a place on a day's route.
type Stop struct {
	Station
	Kind string `json:"kind"` // start, charge, overnight, dest
	// Near is the closest charger to a night stop that is not one itself (a hotel), filled in by the caller.
	Near *Near `json:"near,omitempty"`
}

// Near is a charger close to a stop.
type Near struct {
	Station
	Km float64 `json:"km"`
}

// Day is one day of the trip.
type Day struct {
	Stops       []Stop  `json:"stops"`
	StartPct    float64 `json:"startPct"`
	Km          float64 `json:"km"`
	DriveMin    float64 `json:"driveMin"`
	ChargeMin   float64 `json:"chargeMin"`
	ChargeStops int     `json:"chargeStops"`
}

const (
	bend       = 1.3
	overhead   = 6.0 // minutes to park, plug in and leave
	corridorX  = 1.25
	corridorKm = 25.0
)

// ErrNoChain says no set of stops keeps every leg inside the limits.
var ErrNoChain = errors.New("no chain of chargers keeps every leg inside your limits")

func haversineKm(a, b Station) float64 {
	const r = 6371.0
	p1, p2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dp, dl := p2-p1, (b.Lon-a.Lon)*math.Pi/180
	h := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// Via is a place the trip must pass, in the order given (to take one road and not another).
type Via struct {
	Station
	Charge bool // plug in here: true when it has a charger, false to drive through on what is left
	Night  bool // the day ends here: a hotel already booked
}

type link struct {
	Station
	kind  string
	night bool    // the day must end here
	pct   float64 // the charge on leaving
}

// Plan chooses the chargers from stations, and the days. Each via is passed in order; the trip is planned from one to the next.
func Plan(p Params, stations []Station, vias []Via) ([]Day, error) {
	if p.SpeedKph <= 0 {
		p.SpeedKph = 105
	}
	if p.CarKw <= 0 {
		p.CarKw = 100
	}
	if p.CapacityKwh <= 0 || p.KwhPerKm <= 0 {
		return nil, errors.New("the battery size and efficiency are needed")
	}
	chain := []link{{Station: p.Start, kind: "start", pct: p.StartPct}}
	cur, pct := p.Start, p.StartPct
	for i := 0; i <= len(vias); i++ {
		target, kind := p.Dest, "dest"
		if i < len(vias) {
			target, kind = vias[i].Station, "via"
		}
		seg, lastKm, err := segment(p, cur, target, pct, stations)
		if err != nil {
			return nil, err
		}
		for _, st := range seg[1 : len(seg)-1] {
			chain = append(chain, link{Station: st, kind: "charge", pct: p.TargetPct})
		}
		if kind == "via" && vias[i].Charge {
			kind = "charge"
		}
		from := p.TargetPct
		if len(seg) == 2 {
			from = pct
		}
		pct = from - lastKm*p.KwhPerKm/p.CapacityKwh*100
		if kind == "charge" {
			pct = p.TargetPct
		}
		chain = append(chain, link{Station: target, kind: kind, night: kind != "dest" && vias[i].Night, pct: pct})
		cur = target
	}
	return split(p, chain), nil
}

// segment finds the best chargers between two places: the nodes (start, chargers, end) and the length of the last leg.
func segment(p Params, from, to Station, startPct float64, stations []Station) ([]Station, float64, error) {
	direct := haversineKm(from, to)
	nodes := []Station{from}
	for _, s := range stations {
		if haversineKm(from, s)+haversineKm(s, to) <= direct*corridorX+corridorKm {
			nodes = append(nodes, s)
		}
	}
	nodes = append(nodes, to)
	n := len(nodes)
	toDest := make([]float64, n)
	for i := range nodes {
		toDest[i] = haversineKm(nodes[i], to)
	}
	rng := func(pct float64) float64 { return (pct - p.ReservePct) / 100 * p.CapacityKwh / p.KwhPerKm }
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return toDest[order[a]] > toDest[order[b]] })
	best := make([]float64, n)
	prev := make([]int, n)
	for i := range best {
		best[i], prev[i] = math.Inf(1), -1
	}
	best[0] = 0
	for _, u := range order {
		if math.IsInf(best[u], 1) {
			continue
		}
		reach := rng(p.TargetPct)
		if u == 0 {
			reach = rng(startPct)
		}
		for _, v := range order {
			if v == u || v == 0 || toDest[v] >= toDest[u] {
				continue
			}
			km := haversineKm(nodes[u], nodes[v]) * bend
			m := minutes(p, km)
			if km > reach || (p.MaxLegKm > 0 && km > p.MaxLegKm) || (p.MaxLegMin > 0 && m > p.MaxLegMin) {
				continue
			}
			cost := best[u] + m
			if v != n-1 {
				cost += chargeMin(p, nodes[v]) + nodes[v].Penalty
			}
			if cost < best[v] {
				best[v], prev[v] = cost, u
			}
		}
	}
	if math.IsInf(best[n-1], 1) {
		return nil, 0, ErrNoChain
	}
	var seg []Station
	for v := n - 1; v != -1; v = prev[v] {
		seg = append([]Station{nodes[v]}, seg...)
	}
	return seg, haversineKm(seg[len(seg)-2], seg[len(seg)-1]) * bend, nil
}

func minutes(p Params, km float64) float64 { return km / p.SpeedKph * 60 }

// chargeMin is a typical stop: from the reserve most of the way to the target, at what the charger and the car allow.
func chargeMin(p Params, s Station) float64 {
	kw := p.CarKw
	if s.Kw > 0 && float64(s.Kw) < kw {
		kw = float64(s.Kw)
	}
	return overhead + (p.TargetPct-p.ReservePct)*0.7/100*p.CapacityKwh/kw*60
}

// split cuts the chain into days: when the next leg would pass the day limit, the night is spent at the stop already reached.
func split(p Params, chain []link) []Day {
	day := Day{StartPct: p.StartPct, Stops: []Stop{{Station: chain[0].Station, Kind: "start"}}}
	days := []Day{}
	elapsed := 0.0
	for i := 1; i < len(chain); i++ {
		a, b := chain[i-1], chain[i]
		km := haversineKm(a.Station, b.Station) * bend
		m := minutes(p, km)
		if i > 1 && (a.night || (p.DayMin > 0 && elapsed+m > p.DayMin)) {
			day.Stops[len(day.Stops)-1].Kind = "overnight"
			days = append(days, day)
			day = Day{StartPct: a.pct, Stops: []Stop{{Station: a.Station, Kind: "overnight"}}}
			elapsed = 0
		}
		day.Km += km
		day.DriveMin += m
		elapsed += m
		if b.kind == "charge" {
			c := chargeMin(p, b.Station)
			day.ChargeMin += c
			day.ChargeStops++
			elapsed += c
		}
		day.Stops = append(day.Stops, Stop{Station: b.Station, Kind: b.kind})
	}
	return append(days, day)
}
