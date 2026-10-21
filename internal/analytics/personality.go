package analytics

import (
	"sort"
	"time"
)

// Personality is a playful name for how the car is used, with the fact behind it.
type Personality struct {
	Key   string  `json:"key"`
	Name  string  `json:"name"`
	Emoji string  `json:"emoji"`
	Why   string  `json:"why"`   // the page fills the number in: "{pct} of your charging started after 9 pm"
	Share float64 `json:"share"` // the number behind it, 0 to 1
	score float64
}

// Personalities ranks the ones that fit, best first. Each has a bar to clear; the score is how far past it the car is, as a
// share of the room above the bar, so a low bar does not win by being low.
func Personalities(ts []Trip, cs []Charge, loc *time.Location) []Personality {
	var out []Personality
	add := func(key, name, emoji, why string, share, bar float64) {
		if share >= bar {
			room := 1 - bar
			if room <= 0 {
				room = 0.3 // the hypermiler, whose share is against a line rather than out of one
			}
			out = append(out, Personality{Key: key, Name: name, Emoji: emoji, Why: why, Share: share, score: (share - bar) / room})
		}
	}
	var night, chargeN, dcKwh, kwh float64
	for _, c := range cs {
		if c.Kwh <= 0 {
			continue
		}
		chargeN++
		kwh += c.Kwh
		if hr := time.Unix(c.Start, 0).In(loc).Hour(); hr >= 21 || hr < 6 {
			night++
		}
		if c.PeakKw >= 30 {
			dcKwh += c.Kwh
		}
	}
	if chargeN >= 4 {
		add("night", "The Night Charger", "🌙", "{pct} of your charging started after 9 pm", night/chargeN, 0.6)
	}
	if kwh > 0 && chargeN >= 3 {
		add("fast", "The Fast-Charge Fan", "⚡", "{pct} of your energy came from fast chargers", dcKwh/kwh, 0.4)
	}
	var km, commuteN, tripN, longKm, weekendKm, effKm, effKwh, shortN float64
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		tripN++
		km += t.Km
		st := time.Unix(t.Start, 0).In(loc)
		wd, hr := st.Weekday(), st.Hour()
		if wd != time.Saturday && wd != time.Sunday && ((hr >= 6 && hr < 9) || (hr >= 15 && hr < 19)) {
			commuteN++
		}
		if wd == time.Saturday || wd == time.Sunday {
			weekendKm += t.Km
		}
		if t.Km >= 80 {
			longKm += t.Km
		}
		if t.Km < 10 {
			shortN++
		}
		if t.Kwh > 0 {
			effKm += t.Km
			effKwh += t.Kwh
		}
	}
	if tripN >= 5 {
		add("commuter", "The Commuter", "🏢", "{pct} of your trips were weekday rush hours", commuteN/tripN, 0.45)
		add("road", "The Road Tripper", "🛣️", "{pct} of your miles were on drives of 50 miles or more", longKm/km, 0.3)
		add("weekend", "The Weekend Wanderer", "🌄", "{pct} of your miles were on weekends", weekendKm/km, 0.4)
		add("local", "The Local Hero", "🏘️", "{pct} of your trips were under 6 miles", shortN/tripN, 0.6)
	}
	if effKwh > 0 && effKm >= 100 {
		// 7.2 km/kWh is about 4.5 mi/kWh; the share is how efficient against that line
		add("hyper", "The Hypermiler", "🍃", "{eff} on average", (effKm/effKwh)/7.2, 1)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	if len(out) == 0 && tripN > 0 {
		out = append(out, Personality{Key: "steady", Name: "The Steady Driver", Emoji: "🚗", Why: "a bit of everything, nothing to extremes", Share: 1})
	}
	return out
}
