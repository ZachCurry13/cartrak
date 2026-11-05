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
	// Real, when set, gives the road distance, the driving time and the energy per kilometer between two places once they are known, so a
	// leg is not left as a straight line made longer (which is wrong across water or round a lake) and its energy follows the speed it is
	// driven at. A zero energy means the one in KwhPerKm.
	Real func(a, b Station) (km, min, kwhPerKm float64, ok bool)
}

// leg is the distance and driving time between two places: the real road when known, else an estimate.
func (p Params) leg(a, b Station) (km, min float64) {
	km, min, _ = p.legFull(a, b)
	return km, min
}

// legFull is leg and the energy it takes, in kWh.
func (p Params) legFull(a, b Station) (km, min, kwh float64) {
	if p.Real != nil {
		if km, min, e, ok := p.Real(a, b); ok {
			if e <= 0 {
				e = p.KwhPerKm
			}
			return km, min, km * e
		}
	}
	km = haversineKm(a, b) * bend
	return km, minutes(p, km), km * p.KwhPerKm
}

// Stop is a place on a day's route.
type Stop struct {
	Station
	Kind string `json:"kind"` // start, charge, overnight, dest
	// Near is the closest charger to a night stop that is not one itself (a hotel), filled in by the caller.
	Near *Near `json:"near,omitempty"`
	// The day's timeline at this stop: minutes from the start of the day, the battery on arriving and leaving, and the charge taken.
	ArriveMin float64  `json:"arriveMin"`
	DepartMin float64  `json:"departMin"`
	ArrivePct float64  `json:"arrivePct"`
	DepartPct float64  `json:"departPct"`
	Kwh       float64  `json:"kwh"`            // energy added here
	ChargeMin float64  `json:"chargeMin"`      // time plugged in here
	LegKm     float64  `json:"legKm"`          // the distance of the drive into this stop
	Cost      *float64 `json:"cost,omitempty"` // what the energy added here is expected to cost, set by the caller
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
	DepartTS    int64   `json:"departTs"` // when the day starts, set by the caller
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
		seg, lastKwh, err := segment(p, cur, target, pct, stations)
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
		pct = from - lastKwh/p.CapacityKwh*100
		if kind == "charge" {
			pct = p.TargetPct
		}
		chain = append(chain, link{Station: target, kind: kind, night: kind != "dest" && vias[i].Night, pct: pct})
		cur = target
	}
	return split(p, chain), nil
}

// segment finds the best chargers between two places: the nodes (start, chargers, end) and the energy of the last leg.
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
	avail := func(pct float64) float64 { return (pct - p.ReservePct) / 100 * p.CapacityKwh } // kWh above the reserve
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
		reach := avail(p.TargetPct)
		if u == 0 {
			reach = avail(startPct)
		}
		for _, v := range order {
			if v == u || v == 0 || toDest[v] >= toDest[u] {
				continue
			}
			km, m, kwh := p.legFull(nodes[u], nodes[v])
			if kwh > reach || (p.MaxLegKm > 0 && km > p.MaxLegKm) || (p.MaxLegMin > 0 && m > p.MaxLegMin) {
				continue
			}
			cost := best[u] + m
			if v != n-1 {
				cost += chargeMin(p, nodes[v], kwh) + nodes[v].Penalty
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
	_, _, lastKwh := p.legFull(seg[len(seg)-2], seg[len(seg)-1])
	return seg, lastKwh, nil
}

func minutes(p Params, km float64) float64 { return km / p.SpeedKph * 60 }

// chargeMin is what a stop costs when it replaces the energy the leg into it used: the time to park and plug in, plus that energy
// (kWh) at what the charger and the car allow.
func chargeMin(p Params, s Station, legKwh float64) float64 {
	kw := p.CarKw
	if s.Kw > 0 && float64(s.Kw) < kw {
		kw = float64(s.Kw)
	}
	return overhead + legKwh/kw*60
}

// margin is the percent of battery kept above the reserve on leaving a charger.
const margin = 5.0

// split cuts the chain into days: when the next leg would pass the day limit, the night is spent at the stop already reached.
// It also carries the battery along: a stop charges only as far as the next leg needs (never past the target), at the power of the
// charger or the car, whichever is lower.
func split(p Params, chain []link) []Day {
	drop := func(kwh float64) float64 { return kwh / p.CapacityKwh * 100 }
	day := Day{StartPct: p.StartPct, Stops: []Stop{{Station: chain[0].Station, Kind: "start", DepartPct: p.StartPct, ArrivePct: p.StartPct}}}
	days := []Day{}
	elapsed, pct := 0.0, p.StartPct
	for i := 1; i < len(chain); i++ {
		a, b := chain[i-1], chain[i]
		km, m, kwh := p.legFull(a.Station, b.Station)
		if i > 1 && (a.night || (p.DayMin > 0 && elapsed+m > p.DayMin)) {
			day.Stops[len(day.Stops)-1].Kind = "overnight"
			days = append(days, day)
			if a.kind == "charge" { // a night at a charger fills the battery
				pct = p.TargetPct
			}
			day = Day{StartPct: pct, Stops: []Stop{{Station: a.Station, Kind: "overnight", ArrivePct: pct, DepartPct: pct}}}
			elapsed = 0
		}
		arrive := pct - drop(kwh)
		day.Km += km
		day.DriveMin += m
		elapsed += m
		st := Stop{Station: b.Station, Kind: b.kind, ArriveMin: elapsed, ArrivePct: arrive, DepartPct: arrive, LegKm: km}
		if b.kind == "charge" {
			depart := p.TargetPct
			if i+1 < len(chain) {
				_, _, nextKwh := p.legFull(b.Station, chain[i+1].Station)
				next := drop(nextKwh)
				depart = math.Min(p.TargetPct, math.Max(arrive, p.ReservePct+next+margin))
			}
			kw := p.CarKw
			if b.Kw > 0 && float64(b.Kw) < kw {
				kw = float64(b.Kw)
			}
			st.Kwh = math.Max(0, (depart-arrive)/100*p.CapacityKwh)
			st.ChargeMin = overhead + st.Kwh/kw*60
			st.DepartPct = depart
			day.ChargeMin += st.ChargeMin
			day.ChargeStops++
			elapsed += st.ChargeMin
		}
		st.DepartMin = elapsed
		pct = st.DepartPct
		day.Stops = append(day.Stops, st)
	}
	return append(days, day)
}
