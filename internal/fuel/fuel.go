// Package fuel is the arithmetic for gas and hybrid cars. Ford reports the tank as a percent, so gallons come from a tank size
// CarTrak learns: every logged fill-up says how many gallons took the gauge from one level to another.
package fuel

import "sort"

const (
	// MinRise is how far the gauge must climb between two readings to count as filling up. Smaller moves are the fuel sloshing.
	MinRise = 8.0
	// TankMinRise is the climb a fill-up needs to say anything about the tank size: a small top-up is too coarse a measure.
	TankMinRise = 15.0
	// LitersPerGallon is the US gallon.
	LitersPerGallon = 3.785411784
	// MilesPerKm is how many miles are in a kilometer.
	MilesPerKm = 0.621371
)

// Fill is one time the tank was filled: the gauge before and after, found from the readings, and what was put in, entered by hand.
type Fill struct {
	ID          int64    `json:"id"`
	TS          int64    `json:"ts"`
	StartPct    float64  `json:"startPct"` // the tank before, percent
	EndPct      float64  `json:"endPct"`   // the tank after, percent
	OdoKm       *float64 `json:"odoKm"`
	Gallons     *float64 `json:"gallons"`     // entered by the owner, always US gallons
	PricePerGal *float64 `json:"pricePerGal"` // entered each time
	Status      string   `json:"status"`      // pending (found, nothing entered), logged, or dismissed (not a fill-up)
	Source      string   `json:"source"`      // auto (found from readings) or manual
}

// Cost is what the fill-up cost, when both the gallons and the price are known.
func (f Fill) Cost() *float64 {
	if f.Gallons == nil || f.PricePerGal == nil {
		return nil
	}
	c := *f.Gallons * *f.PricePerGal
	return &c
}

// Rise is how far the gauge climbed between two readings, in percent points, 0 when it did not.
func Rise(before, after *float64) float64 {
	if before == nil || after == nil || *after <= *before {
		return 0
	}
	return *after - *before
}

// Filled says whether a climb looks like filling up.
func Filled(before, after *float64) bool { return Rise(before, after) >= MinRise }

// TankGallons is the size of the tank, worked out from the fill-ups that say how many gallons they took and how far the gauge
// climbed: the middle of the sizes they imply, so one odd gauge does not move it. n is how many fill-ups it rests on, 0 when
// none can say yet.
func TankGallons(fills []Fill) (gallons float64, n int) {
	var sizes []float64
	for _, f := range fills {
		if f.Status == "dismissed" || f.Gallons == nil || *f.Gallons <= 0 {
			continue
		}
		if rise := f.EndPct - f.StartPct; rise >= TankMinRise {
			if g := *f.Gallons / (rise / 100); g >= 3 && g <= 60 { // a gauge that disagrees wildly is not believed
				sizes = append(sizes, g)
			}
		}
	}
	if len(sizes) == 0 {
		return 0, 0
	}
	sort.Float64s(sizes)
	mid := len(sizes) / 2
	if len(sizes)%2 == 0 {
		return (sizes[mid-1] + sizes[mid]) / 2, len(sizes)
	}
	return sizes[mid], len(sizes)
}

// Used is the fuel burned while the gauge fell from one level to another, in gallons. ok is false when the tank is not known
// yet or the gauge did not fall enough to mean anything.
func Used(tankGal, startPct, endPct float64) (gallons float64, ok bool) {
	if tankGal <= 0 || startPct-endPct < 1 {
		return 0, false
	}
	return (startPct - endPct) / 100 * tankGal, true
}

// KmPerTank is how far a full tank would go at the pace of a drive that used a given part of it: distance over the share used.
func KmPerTank(km, startPct, endPct float64) (float64, bool) {
	if startPct-endPct < 2 || km <= 0 {
		return 0, false
	}
	return km / ((startPct - endPct) / 100), true
}

// Between is the economy from one fill-up to the next: the distance driven over the fuel the gauge says was burned in between.
// kmPerGal is 0 when it cannot be told.
func Between(prev, next Fill, tankGal float64) (km, gallons, kmPerGal float64) {
	if prev.OdoKm == nil || next.OdoKm == nil || *next.OdoKm <= *prev.OdoKm {
		return 0, 0, 0
	}
	km = *next.OdoKm - *prev.OdoKm
	g, ok := Used(tankGal, prev.EndPct, next.StartPct)
	if !ok || km < 20 {
		return km, 0, 0
	}
	return km, g, km / g
}
