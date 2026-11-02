// Package routeplan carries the battery through a route of several stops: the energy each leg uses (already worked out from the
// weather expected on it and how the car usually drives in that weather) takes the charge down, and where it would fall below the
// reserve a charging stop is added.
package routeplan

import "math"

// Leg is one stretch between two stops.
type Leg struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Km       float64 `json:"km"`
	Minutes  float64 `json:"minutes"`  // driving time
	KwhPerKm float64 `json:"kwhPerKm"` // the car's expected energy use on this leg, weather included
}

// Input is the route and the battery it starts with.
type Input struct {
	Legs        []Leg
	StartPct    float64
	ReservePct  float64 // the charge to keep in hand
	TargetPct   float64 // the most a charging stop fills to
	CapacityKwh float64
	ChargeKw    float64 // average charging power over a stop
}

// Charge is a stop to plug in before a leg.
type Charge struct {
	FromPct float64 `json:"fromPct"`
	ToPct   float64 `json:"toPct"`
	Kwh     float64 `json:"kwh"`
	Minutes float64 `json:"minutes"`
}

// LegOut is a leg with the battery worked through it.
type LegOut struct {
	Leg
	Kwh       float64 `json:"kwh"`
	DepartPct float64 `json:"departPct"`
	ArrivePct float64 `json:"arrivePct"`
	Charge    *Charge `json:"charge"` // at the start of this leg
	Short     bool    `json:"short"`  // not reachable with the reserve even from a full charge target
}

// Output is the planned route.
type Output struct {
	Legs            []LegOut `json:"legs"`
	TotalKm         float64  `json:"totalKm"`
	DrivingMinutes  float64  `json:"drivingMinutes"`
	ChargingMinutes float64  `json:"chargingMinutes"`
	TotalKwh        float64  `json:"totalKwh"`
	ArrivePct       float64  `json:"arrivePct"`
	Warnings        []string `json:"warnings"`
}

const margin = 2.0 // percent of battery added on top of what the legs ahead need

// Plan works the battery through the legs.
func Plan(in Input) Output {
	out := Output{Legs: []LegOut{}, Warnings: []string{}}
	if in.CapacityKwh <= 0 {
		return out
	}
	drop := func(l Leg) float64 { return l.Km * l.KwhPerKm / in.CapacityKwh * 100 }
	pct := in.StartPct
	for i, l := range in.Legs {
		lo := LegOut{Leg: l, Kwh: l.Km * l.KwhPerKm}
		d := drop(l)
		if pct < in.ReservePct+d && pct < in.TargetPct { // plug in before this leg, for it and as many of the next as fit
			cover := d
			for j := i + 1; j < len(in.Legs) && in.ReservePct+cover+drop(in.Legs[j]) <= in.TargetPct; j++ {
				cover += drop(in.Legs[j])
			}
			to := math.Min(in.TargetPct, in.ReservePct+cover+margin)
			if to > pct {
				kwh := (to - pct) / 100 * in.CapacityKwh
				c := &Charge{FromPct: pct, ToPct: to, Kwh: kwh}
				if in.ChargeKw > 0 {
					c.Minutes = kwh / in.ChargeKw * 60
				}
				lo.Charge = c
				out.ChargingMinutes += c.Minutes
				pct = to
			}
		}
		lo.DepartPct = pct
		pct -= d
		lo.ArrivePct = pct
		if pct < in.ReservePct {
			lo.Short = true
			out.Warnings = append(out.Warnings, "The leg from "+l.From+" to "+l.To+" is more than a charge to "+itoa(in.TargetPct)+"% covers with your "+itoa(in.ReservePct)+"% reserve: find a charger on the way.")
		}
		out.Legs = append(out.Legs, lo)
		out.TotalKm += l.Km
		out.DrivingMinutes += l.Minutes
		out.TotalKwh += lo.Kwh
	}
	out.ArrivePct = pct
	return out
}

func itoa(f float64) string {
	n := int(math.Round(f))
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
