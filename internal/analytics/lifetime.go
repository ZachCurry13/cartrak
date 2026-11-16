package analytics

import (
	"math"
	"strconv"
	"strings"
)

// Lifetime is an estimate of the car's whole life from the odometer and the driving CarTrak has seen: the energy it took, what that
// cost and the gas it saved. Tracked months may be all cold (or all warm), so the energy is a range: from the efficiency of the
// tracked driving to that of mild weather.
type Lifetime struct {
	OdoKm        float64  `json:"odoKm"`        // the odometer now
	BeforeKm     float64  `json:"beforeKm"`     // driven before CarTrak's first trip
	TrackedKm    float64  `json:"trackedKm"`    // the trips CarTrak recorded
	Since        int64    `json:"since"`        // CarTrak's first trip
	KmPerKwh     float64  `json:"kmPerKwh"`     // the year-round efficiency used for the middle figure
	TrackedEff   float64  `json:"trackedEff"`   // what the tracked driving averaged
	MildEff      float64  `json:"mildEff"`      // in mild weather (50 °F and up)
	MildMeasured bool     `json:"mildMeasured"` // MildEff is from your own mild-weather trips, not assumed
	BatteryKwh   float64  `json:"batteryKwh"`   // into the battery over the car's life (middle)
	BatteryLow   float64  `json:"batteryLow"`
	BatteryHigh  float64  `json:"batteryHigh"`
	WallKwh      float64  `json:"wallKwh"` // from chargers, allowing for losses
	Cost         *float64 `json:"cost"`    // at what your tracked charging cost per kWh; nil when nothing was priced
	GasCost      *float64 `json:"gasCost"` // what the same miles would have cost in the gas car you compare with; nil when unknown
	GasLiters    *float64 `json:"gasLiters"`
}

const (
	mildC         = 10.0 // 50 °F: from here up, a battery car is near its best
	mildAssumed   = 1.2  // with no mild-weather trips yet, mild weather is taken as 20% more efficient than the cold tracked months
	defaultWallTo = 1 / 0.9
)

// EstimateLifetime works the estimate out. ok is false until there is enough driving to go on (five trips and 50 km with energy).
func EstimateLifetime(ts []Trip, cs []Charge, odoKm float64) (Lifetime, bool) {
	l := Lifetime{OdoKm: odoKm}
	var km, kwh, mildKm, mildKwh, tempKm, tempSum float64
	n := 0
	first, odoAt, firstOdo := int64(0), int64(0), 0.0
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		l.TrackedKm += t.Km
		if first == 0 || t.Start < first {
			first = t.Start
		}
		if t.OdoKm > 0 && (odoAt == 0 || t.Start < odoAt) { // the odometer where the first trip that knows it began
			odoAt, firstOdo = t.Start, t.OdoKm-t.Km
		}
		if t.Km < 2 || t.Kwh < 0.3 || t.Corrected {
			continue
		}
		km, kwh, n = km+t.Km, kwh+t.Kwh, n+1
		if t.TempC != nil {
			tempKm, tempSum = tempKm+t.Km, tempSum+*t.TempC*t.Km
			if *t.TempC >= mildC {
				mildKm, mildKwh = mildKm+t.Km, mildKwh+t.Kwh
			}
		}
	}
	if n < 5 || km < 50 || odoKm <= 0 {
		return l, false
	}
	l.Since = first
	l.BeforeKm = math.Max(0, firstOdo)
	l.TrackedEff = km / kwh
	switch {
	case mildKm >= 50:
		l.MildEff, l.MildMeasured = mildKm/mildKwh, true
	case tempKm > 0 && tempSum/tempKm < mildC:
		l.MildEff = l.TrackedEff * mildAssumed
	default:
		l.MildEff = l.TrackedEff
	}
	lo, hi := math.Min(l.TrackedEff, l.MildEff), math.Max(l.TrackedEff, l.MildEff)
	l.BatteryLow, l.BatteryHigh = odoKm/hi, odoKm/lo
	l.BatteryKwh = (l.BatteryLow + l.BatteryHigh) / 2
	l.KmPerKwh = odoKm / l.BatteryKwh
	// from the charger: the share lost on the way, and the price, as your tracked charging had them
	var into, wall, paid, pricedWall, gasCost, gasL, gasKwh float64
	for _, c := range cs {
		if c.Kwh > 0 && c.WallKwh > 0 {
			into, wall = into+c.Kwh, wall+c.WallKwh
		}
		if c.Cost != nil && c.WallKwh > 0 {
			paid, pricedWall = paid+*c.Cost, pricedWall+c.WallKwh
		}
		if c.GasOK && c.Kwh > 0 {
			gasCost, gasL, gasKwh = gasCost+c.GasCost, gasL+c.GasLiters, gasKwh+c.Kwh
		}
	}
	ratio := defaultWallTo
	if into >= 20 && wall >= into {
		ratio = wall / into
	}
	l.WallKwh = l.BatteryKwh * ratio
	if pricedWall >= 20 {
		v := l.WallKwh * paid / pricedWall
		l.Cost = &v
	}
	if gasKwh >= 20 {
		c, g := l.BatteryKwh*gasCost/gasKwh, l.BatteryKwh*gasL/gasKwh
		l.GasCost, l.GasLiters = &c, &g
	}
	return l, true
}

// WithLifetime marks the trophies the car had already reached before CarTrak began: the distance ones by the odometer (a fact), the
// energy ones by the low end of the estimate (so an estimate never earns one too early), and adds the estimated energy before
// tracking to the energy ladder's progress.
func WithLifetime(bs []Badge, l Lifetime) []Badge {
	if l.Since == 0 {
		return bs
	}
	before := l.BeforeKm / math.Max(l.TrackedEff, l.MildEff) // energy before tracking, low end
	out := make([]Badge, len(bs))
	copy(out, bs)
	for i := range out {
		b := &out[i]
		switch b.Key {
		case "km160", "km1600", "km16000", "world":
			if b.Earned == 0 && l.BeforeKm+b.Value >= b.Goal {
				b.Value = l.BeforeKm + b.Value
				if l.BeforeKm >= b.Goal {
					b.Earned, b.Note = l.Since, "Already reached when CarTrak began (the odometer)"
				}
			}
		case "mwh1", "mwh10", "mwh100":
			total := before + b.Value
			if b.Earned == 0 && before >= b.Goal {
				b.Earned, b.Note = l.Since, "Reached before CarTrak began (estimated from the odometer)"
			} else if b.Earned == 0 {
				b.Note = "Includes about " + kwhWords(before) + " estimated from the odometer before CarTrak began"
			}
			if b.Earned == 0 || before >= b.Goal {
				b.Value = total
			}
		}
	}
	return out
}

func kwhWords(v float64) string {
	if v >= 1000 {
		return strings.TrimSuffix(strconv.FormatFloat(v/1000, 'f', 1, 64), ".0") + " MWh"
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + " kWh"
}
