package analytics

import (
	"sort"
	"time"
)

// Badge is an achievement: how far along it is and when it was earned. Kind says how to write Value and Goal (km, count,
// kmPerKwh, tempC, kw, kwh, days or towns); for a temperature the goal is a limit to go past, not a total.
type Badge struct {
	Key    string  `json:"key"`
	Name   string  `json:"name"`
	Emoji  string  `json:"emoji"`
	About  string  `json:"about"`
	Kind   string  `json:"kind"`
	Value  float64 `json:"value"`
	Goal   float64 `json:"goal"`
	Earned int64   `json:"earned"`         // when it was earned, 0 when not yet
	Note   string  `json:"note,omitempty"` // said instead of the date, when it was already reached before CarTrak began
	Eta    int64   `json:"eta,omitempty"`  // about when it will be earned at the pace of the recent driving, 0 when it cannot be told
}

// Progress is how far along a badge is, 0 to 1.
func (b Badge) Progress() float64 {
	switch {
	case b.Earned > 0:
		return 1
	case b.Kind == "tempC" && b.Goal < 0: // colder is closer: from 20 °C down to the goal
		return clamp01((20 - b.Value) / (20 - b.Goal))
	case b.Kind == "tempC": // hotter is closer: from 20 °C up to the goal
		return clamp01((b.Value - 20) / (b.Goal - 20))
	case b.Goal > 0:
		return clamp01(b.Value / b.Goal)
	}
	return 0
}

func clamp01(v float64) float64 { return max(0, min(1, v)) }

// tracker follows one badge through the history: a running value, and the moment it reached the goal.
type tracker struct {
	b    Badge
	less bool // reached by going below the goal (cold)
}

// see takes a new value at a moment: the best so far is kept (the highest, or the lowest for a cold badge).
func (t *tracker) see(v float64, at int64) {
	if t.less {
		t.b.Value = min(t.b.Value, v)
	} else {
		t.b.Value = max(t.b.Value, v)
	}
	reached := t.b.Value >= t.b.Goal
	if t.less {
		reached = t.b.Value <= t.b.Goal
	}
	if reached && t.b.Earned == 0 {
		t.b.Earned = at
	}
}

// Badges works out every achievement from the whole history. towns maps each town visited to when it was first visited.
func Badges(ts []Trip, cs []Charge, towns map[string]int64, loc *time.Location) []Badge {
	mk := func(key, name, emoji, about, kind string, goal float64) *tracker {
		return &tracker{b: Badge{Key: key, Name: name, Emoji: emoji, About: about, Kind: kind, Goal: goal}}
	}
	first100, k1, k10, world := mk("km160", "First 100 miles", "🚗", "Drive 100 miles in all", "km", 160.9), mk("km1600", "1,000 miles", "🛞", "Drive 1,000 miles in all", "km", 1609.3),
		mk("km16000", "10,000 miles", "🏆", "Drive 10,000 miles in all", "km", 16093), mk("world", "Around the world", "🌎", "Drive the distance around the Earth", "km", 40075)
	month1000, road, longDay := mk("month1000", "1,000-mile month", "🗓️", "Drive 1,000 miles in one calendar month", "km", 1609.3), mk("road", "Road trip", "🛣️", "One drive of 100 miles or more", "km", 160.9),
		mk("longday", "Long day", "☀️", "Drive 250 miles in one day", "km", 402.3)
	streak30, trips100, trips1000 := mk("streak30", "30-day streak", "🔥", "Drive 30 days in a row", "days", 30), mk("trips100", "100 trips", "💯", "Make 100 trips", "count", 100), mk("trips1000", "1,000 trips", "🎖️", "Make 1,000 trips", "count", 1000)
	hyper, early := mk("hyper", "Hypermiler", "🍃", "A drive of 10 miles or more at 5.5 mi/kWh or better", "kmPerKwh", 8.85), mk("early", "Early bird", "🐦", "Start a drive before 5 am", "count", 1)
	cold, hot := mk("cold", "Sub-zero", "🥶", "Drive when it is below 0 °F outside", "tempC", -17.8), mk("hot", "Heatwave", "🥵", "Drive when it is above 95 °F outside", "tempC", 35)
	cold.less, cold.b.Value, hot.b.Value = true, 20, 20
	club, night, mwh1, mwh10 := mk("kw150", "150 kW club", "⚡", "Charge at 150 kW or more", "kw", 150), mk("night50", "Night owl", "🌙", "Start 50 charges between 9 pm and 6 am", "count", 50),
		mk("mwh1", "1 MWh", "🔋", "Charge 1,000 kWh in all", "kwh", 1000), mk("mwh10", "10 MWh", "🔋", "Charge 10,000 kWh in all", "kwh", 10000)
	explorer := mk("towns25", "Explorer", "🗺️", "Visit 25 different towns", "towns", 25)

	sort.Slice(ts, func(i, j int) bool { return ts[i].Start < ts[j].Start })
	var km float64
	var trips float64
	months, days := map[string]float64{}, map[string]float64{}
	var run float64
	var lastDay time.Time
	ceiling := EffCeiling(ts)
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		km += t.Km
		trips++
		for _, b := range []*tracker{first100, k1, k10, world} {
			b.see(km, t.End)
		}
		trips100.see(trips, t.End)
		trips1000.see(trips, t.End)
		road.see(t.Km, t.End)
		st := time.Unix(t.Start, 0).In(loc)
		months[st.Format("2006-01")] += t.Km
		month1000.see(months[st.Format("2006-01")], t.End)
		dk := st.Format("2006-01-02")
		days[dk] += t.Km
		longDay.see(days[dk], t.End)
		day, _ := time.Parse("2006-01-02", dk)
		switch {
		case lastDay.IsZero() || day.Sub(lastDay) > 24*time.Hour:
			run = 1
		case day.Sub(lastDay) == 24*time.Hour:
			run++
		}
		lastDay = day
		streak30.see(run, t.End)
		if st.Hour() < 5 {
			early.see(1, t.Start)
		}
		if t.Kwh > 0 && t.Km >= 16 && !t.Corrected && t.Km/t.Kwh < ceiling {
			hyper.see(t.Km/t.Kwh, t.End)
		}
		if t.TempC != nil && t.Km >= 2 {
			cold.see(*t.TempC, t.Start)
			hot.see(*t.TempC, t.Start)
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Start < cs[j].Start })
	var kwh, nights float64
	for _, c := range cs {
		kwh += c.Kwh
		mwh1.see(kwh, c.End)
		mwh10.see(kwh, c.End)
		club.see(c.PeakKw, c.Start)
		if hr := time.Unix(c.Start, 0).In(loc).Hour(); hr >= 21 || hr < 6 {
			nights++
			night.see(nights, c.Start)
		}
	}
	firsts := make([]int64, 0, len(towns))
	for _, at := range towns {
		firsts = append(firsts, at)
	}
	sort.Slice(firsts, func(i, j int) bool { return firsts[i] < firsts[j] })
	for i, at := range firsts {
		explorer.see(float64(i+1), at)
	}
	all := []*tracker{first100, k1, k10, world, month1000, road, longDay, streak30, trips100, trips1000, hyper, early, cold, hot, club, night, mwh1, mwh10, explorer}
	out := make([]Badge, len(all))
	for i, t := range all {
		out[i] = t.b
	}
	return out
}
