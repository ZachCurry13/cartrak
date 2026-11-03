// Package chargeeta estimates how long a charge will take from what this car did in its own past sessions: the kind of charger
// (a wall outlet, level 2 or a DC fast charger), how warm the battery was, and how the power fell away near full. It only learns
// from readings CarTrak already keeps; with too little to go on it says so and the caller keeps the plain estimate.
package chargeeta

import (
	"sort"
	"strings"
)

// Level is the kind of charger: "l1" (a wall outlet), "l2" (home or public AC) or "dc" (a fast charger).
type Level string

const (
	L1, L2, DC  Level = "l1", "l2", "dc"
	binPct            = 2.0 // the battery level is split into bins this wide
	minSessions       = 2   // a shape needs this many past sessions of the kind
)

// LevelOf names the charger from the car's own "station type" ("AC basic", "DC") and the power seen. Without a type, over 22 kW is
// taken as DC and under 2.6 kW as a wall outlet.
func LevelOf(powerType string, kw float64) Level {
	t := strings.ToLower(powerType)
	switch {
	case strings.Contains(t, "dc") || kw >= 22:
		return DC
	case kw > 0 && kw < 2.6:
		return L1
	}
	return L2
}

// TempClass puts a battery temperature (°C) in a band that matters for fast charging.
func TempClass(c float64) string {
	switch {
	case c < 10:
		return "cold"
	case c < 20:
		return "cool"
	case c < 38:
		return "warm"
	}
	return "hot"
}

// Obs is one reading of a charge: battery level and power.
type Obs struct{ SoC, Kw float64 }

// Session is one finished charge: the charger kind, the battery and outside temperature at its start (nil when unknown), its readings.
type Session struct {
	Level      Level
	BatC, OutC *float64
	Obs        []Obs
}

// Now is the charge in progress.
type Now struct {
	Level      Level
	SoC, Kw    float64
	BatC, OutC *float64
}

// Result is the estimate and what it rests on.
type Result struct {
	Minutes        float64 `json:"minutes"`
	Learned        bool    `json:"learned"`  // false: the power now carried on to the target, with no learned shape
	Sessions       int     `json:"sessions"` // past sessions the shape rests on
	Level          Level   `json:"level"`
	TempClass      string  `json:"tempClass,omitempty"`
	Preconditioned bool    `json:"preconditioned"` // the battery is warm for the weather: it was warmed before or while plugging in
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if len(s) == 0 {
		return 0
	}
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

func bin(soc float64) int { return min(int(soc/binPct), int(100/binPct)-1) }

// perBin is a session's average power in each bin it has readings for.
func perBin(o []Obs) map[int]float64 {
	sum, n := map[int]float64{}, map[int]int{}
	for _, p := range o {
		if p.Kw > 0.3 {
			sum[bin(p.SoC)] += p.Kw
			n[bin(p.SoC)]++
		}
	}
	out := map[int]float64{}
	for b, s := range sum {
		out[b] = s / float64(n[b])
	}
	return out
}

// shape is the learned power for each bin: for AC the share of the session's own steady power, for DC the power itself. Bins no
// session reached take the value of the nearest lower bin (and 1 for AC, or the first value for DC, below the lowest).
func shape(past []Session, ac bool) ([]float64, int) {
	cols := make([][]float64, int(100/binPct))
	used := 0
	for _, s := range past {
		pb := perBin(s.Obs)
		if len(pb) == 0 {
			continue
		}
		norm := 1.0
		if ac {
			var mid []float64
			for _, p := range s.Obs {
				if p.SoC >= 20 && p.SoC <= 80 && p.Kw > 0.3 {
					mid = append(mid, p.Kw)
				}
			}
			if len(mid) < 3 {
				continue
			}
			norm = median(mid)
		}
		used++
		for b, kw := range pb {
			cols[b] = append(cols[b], kw/norm)
		}
	}
	out := make([]float64, len(cols))
	last := 0.0
	for b := range cols {
		if len(cols[b]) >= 2 {
			last = median(cols[b])
		} else if last == 0 && ac {
			last = 1
		}
		out[b] = last
	}
	for b := range out { // below the first bin that had enough readings, use that bin's value
		if out[b] == 0 {
			for _, v := range out {
				if v > 0 {
					out[b] = v
					break
				}
			}
		}
	}
	return out, used
}

// Estimate is the minutes to reach target (percent) from now, with capKwh the battery's capacity. It returns false when there is no
// charging power to go on.
func Estimate(past []Session, now Now, target, capKwh float64) (Result, bool) {
	if now.Kw < 0.3 || capKwh <= 0 || target <= now.SoC {
		return Result{}, false
	}
	r := Result{Level: now.Level}
	if now.BatC != nil {
		r.TempClass = TempClass(*now.BatC)
		r.Preconditioned = now.OutC != nil && *now.BatC-*now.OutC >= 10 && *now.BatC >= 18
	}
	var match []Session
	for _, s := range past {
		if s.Level == now.Level {
			match = append(match, s)
		}
	}
	if now.Level == DC && now.BatC != nil { // fast charging depends on the battery temperature: prefer sessions in the same band
		var same []Session
		for _, s := range match {
			if s.BatC != nil && TempClass(*s.BatC) == r.TempClass {
				same = append(same, s)
			}
		}
		if len(same) >= minSessions {
			match = same
		}
	}
	ac := now.Level != DC
	sh, used := shape(match, ac)
	step := func(from, to float64, kwAt func(b int) float64) float64 { // hours from one level to another
		var h float64
		for lv := from; lv < to; {
			next := min(to, (float64(bin(lv))+1)*binPct)
			if next <= lv {
				next = to
			}
			kw := max(kwAt(bin(lv)), 0.5)
			h += capKwh * (next - lv) / 100 / kw
			lv = next
		}
		return h
	}
	if used < minSessions { // too little learned: the power now, carried on
		r.Minutes = step(now.SoC, target, func(int) float64 { return now.Kw }) * 60
		return r, true
	}
	r.Learned, r.Sessions = true, used
	here := max(sh[bin(now.SoC)], 1e-6)
	if ac { // the shape is a share of the steady power; the steady power is what the charger gives now
		steady := now.Kw / here
		r.Minutes = step(now.SoC, target, func(b int) float64 { return steady * sh[b] }) * 60
		return r, true
	}
	scale := min(max(now.Kw/here, 0.4), 1.6) // this charger against the learned one at this level
	r.Minutes = step(now.SoC, target, func(b int) float64 { return sh[b] * scale }) * 60
	return r, true
}
