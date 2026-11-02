package weather

import "sort"

// Pt is one reading of a trip that has a position.
type Pt struct {
	TS       int64
	Lat, Lon float64
	Speed    float64 // km/h
}

// Conditions is the rain and snow met while moving on a trip.
type Conditions struct {
	RainMmH   float64 `json:"rainMmH"`   // average rain over the moving time, millimeters an hour
	SnowCmH   float64 `json:"snowCmH"`   // average snowfall over the moving time, centimeters an hour
	PeakMmH   float64 `json:"peakMmH"`   // the heaviest hour met
	WetShare  float64 `json:"wetShare"`  // share of the moving time with rain or snow falling
	Snowy     bool    `json:"snowy"`     // snow was a bigger part of it than rain
	Thunder   bool    `json:"thunder"`   // a thunderstorm hour was met
	Fog       bool    `json:"fog"`       // fog was met (no energy effect is claimed for it)
	Label     string  `json:"label"`     // "Light rain", "Snow", "Dry"
	BasePct   float64 `json:"basePct"`   // the extra energy this is worth with the fixed estimates, before any learning from the owner's trips
	Dry       bool    `json:"dry"`       // nothing fell
	MovingMin float64 `json:"movingMin"` // minutes of moving time the answer rests on
}

// The fixed estimates of how much extra energy a wet road costs, by how hard it falls: spray and a wet surface add rolling
// resistance and drag, and the heater and demister work harder. They are rough, and replaced by what the owner's own wet
// and dry trips show once there are enough of them (see LearnRain).
func rainPct(mmH float64) float64 {
	switch {
	case mmH < 0.1:
		return 0
	case mmH < 0.5:
		return 3
	case mmH < 2.5:
		return 5
	case mmH < 7.5:
		return 8
	}
	return 12
}

func snowPct(cmH float64) float64 {
	switch {
	case cmH < 0.1:
		return 0
	case cmH < 1:
		return 8
	}
	return 15
}

// Summarize walks the moving stretches of a trip and adds up the rain and snow met on each, or false when the weather could
// not be looked up for any of it.
func Summarize(pts []Pt, look func(lat, lon float64, ts int64) (Hour, bool)) (Conditions, bool) {
	var c Conditions
	var sec, rain, snow, wet, pct float64
	worst := 0
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		dt := float64(b.TS - a.TS)
		if dt <= 0 || dt > 300 || a.Speed < 3 || b.Speed < 3 {
			continue
		}
		h, ok := look((a.Lat+b.Lat)/2, (a.Lon+b.Lon)/2, (a.TS+b.TS)/2)
		if !ok || !h.Wx {
			continue
		}
		sec += dt
		rain += h.RainMm * dt
		snow += h.SnowCm * dt
		if h.RainMm > c.PeakMmH {
			c.PeakMmH = h.RainMm
		}
		if h.RainMm >= 0.1 || h.SnowCm >= 0.1 {
			wet += dt
		}
		p := rainPct(h.RainMm)
		if sp := snowPct(h.SnowCm); sp > p {
			p = sp
		}
		pct += p * dt
		if h.Code >= 95 {
			c.Thunder = true
		}
		if h.Code == 45 || h.Code == 48 {
			c.Fog = true
		}
		if h.Code > worst {
			worst = h.Code
		}
	}
	if sec < 60 {
		return Conditions{}, false
	}
	c.RainMmH, c.SnowCmH, c.WetShare, c.BasePct, c.MovingMin = rain/sec, snow/sec, wet/sec, pct/sec, sec/60
	c.Dry = c.WetShare < 0.05
	// Snow melts into the precipitation figure, so judge by the snowfall itself: more than a trace means snow.
	c.Snowy = c.SnowCmH >= 0.1
	c.Label = label(c)
	return c, true
}

func label(c Conditions) string {
	switch {
	case c.Thunder:
		return "Thunderstorm"
	case c.Dry && c.Fog:
		return "Fog"
	case c.Dry:
		return "Dry"
	case c.Snowy && c.SnowCmH >= 1:
		return "Heavy snow"
	case c.Snowy:
		return "Snow"
	case c.RainMmH < 0.5:
		return "Drizzle"
	case c.RainMmH < 2.5:
		return "Light rain"
	case c.RainMmH < 7.5:
		return "Rain"
	}
	return "Heavy rain"
}

// Effect is the extra energy the weather cost, in percent: the fixed estimate times a scale learned from the owner's own trips
// (1 when nothing is learned yet).
func (c Conditions) Effect(scale float64) float64 {
	if scale <= 0 {
		scale = 1
	}
	return c.BasePct * scale
}

// RainSample is one past trip: how hard it rained on average and how its efficiency compared with what its speed and
// temperature predicted (above 1 is better than predicted).
type RainSample struct {
	RainMmH float64
	Ratio   float64
}

// Learned is what the owner's own wet and dry trips say.
type Learned struct {
	WetTrips  int     `json:"wetTrips"`
	DryTrips  int     `json:"dryTrips"`
	EffectPct float64 `json:"effectPct"` // extra energy on a typical wet trip, measured
	Scale     float64 `json:"scale"`     // measured over the fixed estimate for such a trip
}

const (
	wetFrom       = 0.5 // mm an hour on average: a wet trip
	dryTo         = 0.05
	minEach       = 6   // wet and dry trips needed before trusting the measurement
	typicalWetPct = 5.0 // the fixed estimate for light rain, which an average wet trip is closest to
)

// LearnRain compares the efficiency of the owner's wet trips with their dry ones. It needs several of each, and says nothing
// otherwise.
func LearnRain(samples []RainSample) (Learned, bool) {
	var wet, dry []float64
	for _, s := range samples {
		if s.Ratio <= 0 {
			continue
		}
		switch {
		case s.RainMmH >= wetFrom:
			wet = append(wet, s.Ratio)
		case s.RainMmH < dryTo:
			dry = append(dry, s.Ratio)
		}
	}
	if len(wet) < minEach || len(dry) < minEach {
		return Learned{WetTrips: len(wet), DryTrips: len(dry)}, false
	}
	eff := (median(dry)/median(wet) - 1) * 100 // wetter trips go fewer kilometers on a kWh, so dry over wet is above 1
	if eff < 0 {
		eff = 0 // a wet run that was better than dry says the sample is noise, not that rain helps
	}
	if eff > 40 {
		eff = 40
	}
	scale := eff / typicalWetPct
	if scale < 0.25 {
		scale = 0.25
	}
	if scale > 4 {
		scale = 4
	}
	return Learned{WetTrips: len(wet), DryTrips: len(dry), EffectPct: eff, Scale: scale}, true
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}
