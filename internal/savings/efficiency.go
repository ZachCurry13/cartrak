package savings

import "sort"

const (
	day         = 86400
	windowDays  = 30
	minWindowKm = 50.0 // below this much driving the window is too thin to trust, and it widens
	minAnyKm    = 5.0
)

// Drive is a finished trip with the energy it used.
type Drive struct {
	TS  int64
	Km  float64
	Kwh float64
}

// Efficiency answers "how far does a kWh take this car" as of any moment, from the trips before it.
type Efficiency struct{ drives []Drive }

// NewEfficiency keeps the drives that have both a distance and an energy, in time order.
func NewEfficiency(ds []Drive) *Efficiency {
	e := &Efficiency{}
	for _, d := range ds {
		if d.Km > 0.5 && d.Kwh > 0.05 {
			e.drives = append(e.drives, d)
		}
	}
	sort.Slice(e.drives, func(i, j int) bool { return e.drives[i].TS < e.drives[j].TS })
	return e
}

// KmPerKwh is the car's distance per kWh over the 30 days before ts. With too little driving in that window it uses
// everything before ts, and with nothing before ts (a charge at the very start) the first drives after it. Zero when the
// car has no usable trips at all.
func (e *Efficiency) KmPerKwh(ts int64) float64 {
	sum := func(from, to int64) (km, kwh float64) {
		for _, d := range e.drives {
			if d.TS >= from && d.TS < to {
				km += d.Km
				kwh += d.Kwh
			}
		}
		return
	}
	for _, w := range [][2]int64{{ts - windowDays*day, ts}, {0, ts}} {
		if km, kwh := sum(w[0], w[1]); km >= minWindowKm && kwh > 0 {
			return km / kwh
		}
	}
	if km, kwh := sum(0, 1<<62); km >= minAnyKm && kwh > 0 {
		return km / kwh
	}
	return 0
}
