package tripwhy

import (
	"fmt"
	"sort"
)

// BuildWithWind is Build with the wind met on the trip, when it is known.
func BuildWithWind(rs []Reading, tripKwh float64, base Baseline, tempGroup string, regenKm *float64, wind *Wind) (r Report) {
	r.Facts.RegenKm = regenKm
	r.Facts.Wind = wind
	sort.Slice(rs, func(i, j int) bool { return rs[i].TS < rs[j].TS })
	var km, movingKm, movingSec, stoppedSec, totalSec, over100, expectedKwh float64
	var tempSum float64
	var tempN int
	stops := 0
	var idleKwh float64
	hardBrakes, quickStarts := 0, 0
	wasMoving := false
	for i, x := range rs {
		if x.OutsideC != nil {
			tempSum += *x.OutsideC
			tempN++
		}
		if x.Speed > r.Facts.MaxKph {
			r.Facts.MaxKph = x.Speed
		}
		moving := x.Speed >= 3
		if i > 0 && moving != wasMoving && !moving {
			stops++
		}
		wasMoving = moving
		if i == 0 {
			continue
		}
		a := rs[i-1]
		dt := float64(x.TS - a.TS)
		if dt <= 0 || dt > 300 {
			continue
		}
		totalSec += dt
		if dt <= 45 {
			if dv := x.Speed - a.Speed; dv <= -25 {
				hardBrakes++
			} else if dv >= 25 {
				quickStarts++
			}
		}
		if a.Speed < 3 && x.Speed < 3 {
			stoppedSec += dt
			if used := a.Kwh - x.Kwh; used > 0 {
				idleKwh += used
			}
			continue
		}
		d := x.Odo - a.Odo
		if d < 0 || d > dt/3600*250 {
			continue
		}
		avg := (a.Speed + x.Speed) / 2
		km += d
		movingKm += d
		movingSec += dt
		if avg >= 100 {
			over100 += d
		}
		if k := bandIndex(base.Bands, avg); k >= 0 && base.Bands[k].KmPerKwh > 0 {
			expectedKwh += d / base.Bands[k].KmPerKwh
		} else if base.Overall > 0 {
			expectedKwh += d / base.Overall
		}
	}
	r.Facts.Km = km
	if movingSec > 0 {
		r.Facts.AvgKph = movingKm / (movingSec / 3600)
	}
	if km > 0 {
		r.Facts.Share100 = over100 / km
	}
	if totalSec > 0 {
		r.Facts.StoppedPct = stoppedSec / totalSec * 100
	}
	r.Facts.Stops = stops
	r.Facts.IdleKwh, r.Facts.HardBrakes, r.Facts.QuickStarts = idleKwh, hardBrakes, quickStarts
	if regenKm != nil && km > 0 && tripKwh > 0 {
		k := *regenKm * tripKwh / km // the range braking gave back, at what this trip used per kilometer
		r.Facts.RegenKwh = &k
	}
	defer func() { r.Tags = tagsFor(r, km, tripKwh) }()
	if tempN > 0 {
		t := tempSum / float64(tempN)
		r.Facts.TempC = &t
		r.Facts.TempGroup = tempGroup
	}
	if km < 3 || tripKwh <= 0 || base.Overall <= 0 {
		return r
	}
	r.Ready = true
	r.KwhPerKm = tripKwh / km
	r.UsualKwhPerKm = 1 / base.Overall
	r.VsUsualPct = (r.KwhPerKm/r.UsualKwhPerKm - 1) * 100
	speedPct := (expectedKwh/km/r.UsualKwhPerKm - 1) * 100
	tempPct := 0.0
	if e, ok := base.TempKmPKwh[tempGroup]; ok && e > 0 && base.TempOverall > 0 {
		tempPct = (base.TempOverall/e - 1) * 100
	}
	windPct := 0.0
	if wind != nil {
		windPct = wind.EffectPct
	}
	expected := r.UsualKwhPerKm * (1 + speedPct/100) * (1 + tempPct/100) * (1 + windPct/100)
	otherPct := (r.KwhPerKm/expected - 1) * 100
	add := func(key string, pct float64, note string) {
		if pct >= significant || pct <= -significant {
			r.Factors = append(r.Factors, Factor{Key: key, Pct: pct, Note: note})
		}
	}
	add("speed", speedPct, "The speeds driven, compared with your usual mix")
	add("temperature", tempPct, "The outside temperature")
	if wind != nil {
		add("wind", windPct, "The wind along the way")
	}
	if wind != nil {
		add("other", otherPct, "Everything else: hills, heating or cooling the cabin, braking and how you accelerate")
	} else {
		add("other", otherPct, "Everything else: wind, hills, heating or cooling the cabin, braking and how you accelerate")
	}
	sort.Slice(r.Factors, func(i, j int) bool { return abs(r.Factors[i].Pct) > abs(r.Factors[j].Pct) })
	return r
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func sgn(p float64) string {
	if p < 0 {
		return fmt.Sprintf("−%.0f%%", -p)
	}
	return fmt.Sprintf("+%.0f%%", p)
}

func mk(key, label, level string) Tag {
	return Tag{Key: key, Label: label, Level: level, Bad: level == "bad"}
}
