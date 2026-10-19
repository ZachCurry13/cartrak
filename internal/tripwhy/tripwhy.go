// Package tripwhy explains why a trip used more or less energy than usual: how the speed mix, the temperature and
// stopping compare with the driver's own history, and how much is left over (wind, hills, heating and cooling).
package tripwhy

import (
	"fmt"
	"sort"
)

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
// Wind is the wind met on the trip (from the optional weather lookup).
type Wind struct {
	HeadKph   float64 `json:"headKph"` // positive is a headwind, negative a tailwind
	CrossKph  float64 `json:"crossKph"`
	GustKph   float64 `json:"gustKph"`
	EffectPct float64 `json:"effectPct"` // rough change in energy used because of it
}

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
	Wind        *Wind    `json:"wind"`
	RegenKm     *float64 `json:"regenKm"` // range the car says braking gave back during the trip (its own counter)
}

// Tag is one short reason shown next to a trip. Bad means it cost energy.
type Tag struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Bad   bool   `json:"bad"`   // cost this trip energy
	Level string `json:"level"` // "bad" (cost energy), "good" (saved energy) or "info" (describes the trip, no clear effect)
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
func Build(rs []Reading, tripKwh float64, base Baseline, tempGroup string, regenKm *float64) Report {
	return BuildWithWind(rs, tripKwh, base, tempGroup, regenKm, nil)
}

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

// tagsFor picks the short reasons worth showing. A reason is only called bad when the trip really used more energy than
// usual, and good when the numbers say it saved some: a short city trip full of stops and braking can be one of your
// most efficient, so there those are just a description (info), not a culprit.
func tagsFor(r Report, km, tripKwh float64) []Tag {
	var tags []Tag
	f := r.Facts
	worse := r.Ready && r.VsUsualPct > 3
	pct := func(key string) (float64, bool) {
		for _, x := range r.Factors {
			if x.Key == key {
				return x.Pct, true
			}
		}
		return 0, false
	}
	if p, ok := pct("speed"); ok {
		switch {
		case p >= 5:
			tags = append(tags, mk("speed", "Highway speed "+sgn(p), "bad"))
		case p <= -5:
			tags = append(tags, mk("lowspeed", "Low speeds "+sgn(p), "good"))
		}
	}
	if p, ok := pct("temperature"); ok {
		hot := f.TempC != nil && *f.TempC >= 24
		switch {
		case p >= 5 && hot:
			tags = append(tags, mk("hot", "Hot weather "+sgn(p), "bad"))
		case p >= 5:
			tags = append(tags, mk("cold", "Cold weather "+sgn(p), "bad"))
		case p <= -5:
			tags = append(tags, mk("mild", "Mild weather "+sgn(p), "good"))
		}
	}
	if p, ok := pct("wind"); ok && f.Wind != nil {
		switch {
		case p >= 4:
			tags = append(tags, mk("headwind", "Headwind "+sgn(p), "bad"))
		case p <= -4:
			tags = append(tags, mk("tailwind", "Tailwind "+sgn(p), "good"))
		}
	}
	if f.Wind != nil && (f.Wind.CrossKph >= 25 || f.Wind.GustKph >= 55) {
		tags = append(tags, mk("crosswind", "Strong crosswind", "info"))
	}
	level := "info"
	if worse {
		level = "bad"
	}
	if km > 0 && (f.StoppedPct >= 10 || float64(f.Stops)/km*10 >= 3) {
		tags = append(tags, mk("stopgo", "Stop and go", level))
	}
	if tripKwh > 0 && f.IdleKwh >= 0.4 && f.IdleKwh/tripKwh >= 0.04 {
		tags = append(tags, mk("idle", "Idling and climate control", level))
	}
	if km > 0 && float64(f.HardBrakes) >= maxf(3, km/8) {
		tags = append(tags, mk("brake", "Hard braking", level))
	}
	if km > 0 && float64(f.QuickStarts) >= maxf(3, km/8) {
		tags = append(tags, mk("launch", "Quick starts", level))
	}
	if f.RegenKm != nil && km > 0 && *f.RegenKm/km >= 0.04 {
		tags = append(tags, mk("regen", "Regenerative braking helped", "good"))
	}
	if r.Ready && r.VsUsualPct <= -3 && len(tags) == 0 {
		tags = append(tags, mk("gentle", "Better than your usual "+sgn(r.VsUsualPct), "good"))
	}
	return tags
}
