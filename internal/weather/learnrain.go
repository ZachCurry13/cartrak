package weather

import "sort"

// What rain costs this car, learned from past trips: the samples, the fit, and the median that makes the fit robust.
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
