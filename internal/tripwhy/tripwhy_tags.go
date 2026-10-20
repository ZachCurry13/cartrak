package tripwhy

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
