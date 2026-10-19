// Package speedeff works out how much energy the car uses at each speed, from the stored readings of real drives.
package speedeff

import "sort"

// Reading is one stored reading of a moving car.
type Reading struct {
	TS    int64
	Odo   float64 // kilometres
	Kwh   float64 // energy left in the battery
	Speed float64 // km/h
}

// Band is a speed range and what driving in it used.
type Band struct {
	Lo, Hi   float64 // km/h
	Km, Kwh  float64
	KmPerKwh float64
}

// Out is a band as it is shown.
type Out struct {
	LoKph    float64 `json:"loKph"`
	HiKph    float64 `json:"hiKph"`
	Km       float64 `json:"km"`
	Kwh      float64 `json:"kwh"`
	KmPerKwh float64 `json:"kmPerKwh"`
}

var edges = []float64{3, 50, 80, 100, 115, 130, 400}

const (
	maxGap = 120 // seconds between two readings that still count as one stretch of driving
	minKm  = 5.0 // a band needs this much driving before it means anything
)

// Build pairs each reading with the one before it and adds the stretch to the band of its average speed. Pairs
// across a stop, a gap or a charge are left out, so the totals are just driving. Energy can go up (braking recovers
// some), which is why the sum is the net energy used.
func Build(rs []Reading) []Out {
	sort.Slice(rs, func(i, j int) bool { return rs[i].TS < rs[j].TS })
	bands := make([]Band, len(edges)-1)
	for i := range bands {
		bands[i].Lo, bands[i].Hi = edges[i], edges[i+1]
	}
	for i := 1; i < len(rs); i++ {
		a, b := rs[i-1], rs[i]
		dt := b.TS - a.TS
		if dt <= 0 || dt > maxGap || a.Speed < edges[0] || b.Speed < edges[0] {
			continue
		}
		km := b.Odo - a.Odo
		if km < 0 || km > float64(dt)/3600*250 {
			continue
		}
		used := a.Kwh - b.Kwh
		if used < -2 || used > 20 { // a charge or a bad reading, not driving
			continue
		}
		avg := (a.Speed + b.Speed) / 2
		for j := range bands {
			if avg >= bands[j].Lo && avg < bands[j].Hi {
				bands[j].Km += km
				bands[j].Kwh += used
				break
			}
		}
	}
	var out []Out
	for _, b := range bands {
		if b.Km < minKm || b.Kwh <= 0 {
			continue
		}
		out = append(out, Out{LoKph: b.Lo, HiKph: b.Hi, Km: b.Km, Kwh: b.Kwh, KmPerKwh: b.Km / b.Kwh})
	}
	return out
}
