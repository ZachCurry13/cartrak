package weather

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
	Dry       bool    `json:"dry"`       // nothing fell, and the road was not wet from rain just before
	Damp      bool    `json:"damp"`      // dry now but the road was wet from rain in the hours before
	SunWm2    float64 `json:"sunWm2"`    // average sunshine on the ground while moving
	Sun       string  `json:"sun"`       // "Sunny", "Some sun", or "" (night and overcast)
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

// dampPct is what a road still wet from earlier rain costs: about the same as a drizzle falling.
const dampPct = 2.0

// dampFromMm is the rain, added up over the three hours before, that leaves a road wet: even a drizzle of a fifth of a millimeter does.
const dampFromMm = 0.2

// HourEffectPct is the extra energy, in percent, one hour of weather is worth with the fixed estimates: the larger of rain and snow.
func HourEffectPct(h Hour) float64 {
	p := rainPct(h.RainMm)
	if sp := snowPct(h.SnowCm); sp > p {
		return sp
	}
	return p
}

// Summarize walks the moving stretches of a trip and adds up the rain and snow met on each, or false when the weather could
// not be looked up for any of it.
func Summarize(pts []Pt, look func(lat, lon float64, ts int64) (Hour, bool)) (Conditions, bool) {
	var c Conditions
	var sec, rain, snow, wet, pct, damp, sun float64
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
		p := HourEffectPct(h)
		if p == 0 { // nothing falling now: is the road still wet from the hours before?
			mid := (a.TS + b.TS) / 2
			var before float64
			for _, back := range []int64{3600, 7200, 10800} {
				if e, ok := look((a.Lat+b.Lat)/2, (a.Lon+b.Lon)/2, mid-back); ok {
					before += e.RainMm
				}
			}
			if before >= dampFromMm {
				p = dampPct
				damp += dt
			}
		}
		pct += p * dt
		sun += h.Sun * dt
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
	c.Damp = damp/sec >= 0.3
	c.Dry = c.WetShare < 0.05 && !c.Damp
	c.SunWm2 = sun / sec
	switch {
	case c.SunWm2 >= 450:
		c.Sun = "Sunny"
	case c.SunWm2 >= 150:
		c.Sun = "Some sun"
	}
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
	case c.WetShare < 0.05 && c.Damp:
		return "Wet roads"
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
