// Package tripwhy explains why a trip used more or less energy than usual: how the speed mix, the temperature and
// stopping compare with the driver's own history, and how much is left over (wind, hills, heating and cooling).
package tripwhy

import "sort"

// Reading is one stored reading during the trip.
type Reading struct {
	TS       int64
	Odo      float64 // km
	Kwh      float64 // energy left
	Speed    float64 // km/h
	OutsideC *float64
}

// Speed band history: kilometres per kWh for a speed range.
type Band struct {
	LoKph, HiKph float64
	KmPerKwh     float64
}

// Baseline is what the driver's history says.
type Baseline struct {
	Bands       []Band             // by speed
	Overall     float64            // km per kWh over all that driving
	TempKmPKwh  map[string]float64 // km per kWh by temperature group name
	TempOverall float64
}

// Factor is one reason. Pct is how much more (+) or less (-) energy it made the trip use.
type Factor struct {
	Key  string  `json:"key"`
	Pct  float64 `json:"pct"`
	Note string  `json:"note"`
}

// Facts are plain numbers about the trip, for the screen to word in the reader's units.
type Facts struct {
	Km          float64  `json:"km"`
	AvgKph      float64  `json:"avgKph"` // while moving
	MaxKph      float64  `json:"maxKph"`
	Share100    float64  `json:"share100"` // share of the distance above 100 km/h
	Stops       int      `json:"stops"`
	StoppedPct  float64  `json:"stoppedPct"` // share of the trip's time spent stopped
	TempC       *float64 `json:"tempC"`
	TempGroup   string   `json:"tempGroup"`
	IdleKwh     float64  `json:"idleKwh"`     // energy used while stopped (climate control, systems)
	HardBrakes  int      `json:"hardBrakes"`  // big drops in speed between two readings
	QuickStarts int      `json:"quickStarts"` // big gains in speed between two readings
}

// Tag is one short reason shown next to a trip. Bad means it cost energy.
type Tag struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Bad   bool   `json:"bad"`
}

// Report is the explanation.
type Report struct {
	Ready         bool     `json:"ready"`
	KwhPerKm      float64  `json:"kwhPerKm"`
	UsualKwhPerKm float64  `json:"usualKwhPerKm"`
	VsUsualPct    float64  `json:"vsUsualPct"`
	Factors       []Factor `json:"factors"`
	Tags          []Tag    `json:"tags"`
	Facts         Facts    `json:"facts"`
}

const significant = 3.0 // percent: smaller effects are not worth a line

func bandIndex(b []Band, kph float64) int {
	for i := range b {
		if kph >= b[i].LoKph && kph < b[i].HiKph {
			return i
		}
	}
	return -1
}

// Build explains a trip. tempGroup is the name of the temperature group of the trip's average temperature.
func Build(rs []Reading, tripKwh float64, base Baseline, tempGroup string) (r Report) {
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
	expected := r.UsualKwhPerKm * (1 + speedPct/100) * (1 + tempPct/100)
	otherPct := (r.KwhPerKm/expected - 1) * 100
	add := func(key string, pct float64, note string) {
		if pct >= significant || pct <= -significant {
			r.Factors = append(r.Factors, Factor{Key: key, Pct: pct, Note: note})
		}
	}
	add("speed", speedPct, "The speeds driven, compared with your usual mix")
	add("temperature", tempPct, "The outside temperature")
	add("other", otherPct, "Everything else: wind, hills, heating or cooling the cabin, braking and how you accelerate")
	sort.Slice(r.Factors, func(i, j int) bool { return abs(r.Factors[i].Pct) > abs(r.Factors[j].Pct) })
	return r
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// tagsFor picks the short reasons worth showing: the ones that cost energy first, then a good word if nothing did.
func tagsFor(r Report, km, tripKwh float64) []Tag {
	var tags []Tag
	f := r.Facts
	if f.TempC != nil {
		switch {
		case *f.TempC < 0:
			tags = append(tags, Tag{"cold", "Freezing weather", true})
		case *f.TempC < 8:
			tags = append(tags, Tag{"cold", "Cold weather", true})
		case *f.TempC >= 30:
			tags = append(tags, Tag{"hot", "Hot weather", true})
		}
	}
	for _, x := range r.Factors {
		if x.Key == "speed" && x.Pct >= 5 {
			tags = append(tags, Tag{"speed", "Highway speed", true})
		}
	}
	if km > 0 && (f.StoppedPct >= 10 || float64(f.Stops)/km*10 >= 3) {
		tags = append(tags, Tag{"stopgo", "Stop and go", true})
	}
	if tripKwh > 0 && f.IdleKwh >= 0.4 && f.IdleKwh/tripKwh >= 0.04 {
		tags = append(tags, Tag{"idle", "Idling and climate control", true})
	}
	if km > 0 && float64(f.HardBrakes) >= maxf(3, km/8) {
		tags = append(tags, Tag{"brake", "Hard braking", true})
	}
	if km > 0 && float64(f.QuickStarts) >= maxf(3, km/8) {
		tags = append(tags, Tag{"launch", "Quick starts", true})
	}
	if len(tags) == 0 && r.Ready && r.VsUsualPct <= -3 {
		tags = append(tags, Tag{"gentle", "Gentle driving", false})
		if f.TempC != nil && *f.TempC >= 10 && *f.TempC < 28 {
			tags = append(tags, Tag{"mild", "Mild weather", false})
		}
	}
	return tags
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
