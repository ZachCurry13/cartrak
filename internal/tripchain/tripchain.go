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

// Plan chooses the chargers from stations, and the days.
func Plan(p Params, stations []Station) ([]Day, error) {
	if p.SpeedKph <= 0 {
		p.SpeedKph = 105
	}
	if p.CarKw <= 0 {
		p.CarKw = 100
	}
	if p.CapacityKwh <= 0 || p.KwhPerKm <= 0 {
		return nil, errors.New("the battery size and efficiency are needed")
	}
	direct := haversineKm(p.Start, p.Dest)
	nodes := []Station{p.Start}
	for _, s := range stations {
		if haversineKm(p.Start, s)+haversineKm(s, p.Dest) <= direct*corridorX+corridorKm {
			nodes = append(nodes, s)
		}
	}
	nodes = append(nodes, p.Dest)
	n := len(nodes)
	toDest := make([]float64, n)
	for i := range nodes {
		toDest[i] = haversineKm(nodes[i], p.Dest)
	}
	rng := func(pct float64) float64 { return (pct - p.ReservePct) / 100 * p.CapacityKwh / p.KwhPerKm }
	minutes := func(km float64) float64 { return km / p.SpeedKph * 60 }
	chargeMin := func(s Station) float64 { // a typical stop: from the reserve most of the way to the target
		kw := p.CarKw
		if s.Kw > 0 && float64(s.Kw) < kw {
			kw = float64(s.Kw)
		}
		return overhead + (p.TargetPct-p.ReservePct)*0.7/100*p.CapacityKwh/kw*60
	}
	// nodes in order of distance still to go, so a leg only ever goes forward
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
			reach = rng(p.StartPct)
		}
		for _, v := range order {
			if v == u || v == 0 || toDest[v] >= toDest[u] {
				continue
			}
			km := haversineKm(nodes[u], nodes[v]) * bend
			m := minutes(km)
			if km > reach || (p.MaxLegKm > 0 && km > p.MaxLegKm) || (p.MaxLegMin > 0 && m > p.MaxLegMin) {
				continue
			}
			cost := best[u] + m
			if v != n-1 {
				cost += chargeMin(nodes[v])
			}
			if cost < best[v] {
				best[v], prev[v] = cost, u
			}
		}
	}
	if math.IsInf(best[n-1], 1) {
		return nil, ErrNoChain
	}
	var chain []int
	for v := n - 1; v != -1; v = prev[v] {
		chain = append([]int{v}, chain...)
	}
	return split(p, nodes, chain, minutes, chargeMin), nil
}

// split cuts the chain into days: when the next leg would pass the day limit, the night is spent at the charger already reached.
func split(p Params, nodes []Station, chain []int, minutes func(float64) float64, chargeMin func(Station) float64) []Day {
	day := Day{StartPct: p.StartPct, Stops: []Stop{{Station: nodes[chain[0]], Kind: "start"}}}
	days := []Day{}
	elapsed := 0.0
	for i := 1; i < len(chain); i++ {
		a, b := nodes[chain[i-1]], nodes[chain[i]]
		km := haversineKm(a, b) * bend
		m := minutes(km)
		if p.DayMin > 0 && i > 1 && elapsed+m > p.DayMin {
			day.Stops[len(day.Stops)-1].Kind = "overnight"
			days = append(days, day)
			day = Day{StartPct: p.TargetPct, Stops: []Stop{{Station: a, Kind: "overnight"}}}
			elapsed = 0
		}
		day.Km += km
		day.DriveMin += m
		elapsed += m
		if i == len(chain)-1 {
			day.Stops = append(day.Stops, Stop{Station: b, Kind: "dest"})
			continue
		}
		c := chargeMin(b)
		day.ChargeMin += c
		day.ChargeStops++
		elapsed += c
		day.Stops = append(day.Stops, Stop{Station: b, Kind: "charge"})
	}
	return append(days, day)
}
