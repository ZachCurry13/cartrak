package analytics

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

// More trophies, from habits rather than totals: efficiency, charging, places, streaks, odometer milestones and fuel. They use only
// the odometer's own distances (a trip whose distance was corrected never counts toward an efficiency trophy).

// EVOnly are the trophies about charging and battery efficiency; FuelOnly are about the tank. A car that cannot plug in is not shown
// the first, an electric car is not shown the second.
var (
	EVOnly   = map[string]bool{"kw150": true, "mwh1": true, "mwh10": true, "night50": true, "hyper": true, "gentle": true, "winter": true, "lowslow": true, "quick": true, "detective": true, "plugstreak": true, "skipper": true, "mwh100": true}
	FuelOnly = map[string]bool{"muncher": true, "thrifty": true}
)

func newBadge(key, name, emoji, about, kind string, goal float64) *tracker {
	return &tracker{b: Badge{Key: key, Name: name, Emoji: emoji, About: about, Kind: kind, Goal: goal}}
}

// runOf is the longest run of consecutive steps in a sorted list of numbers.
func runOf(vals []int) int {
	best, run := 0, 0
	for i, v := range vals {
		if i > 0 && v == vals[i-1]+1 {
			run++
		} else {
			run = 1
		}
		best = max(best, run)
	}
	return best
}

const kmPerMile = 1.609344

// firstOdo is the odometer at the end of the first trip that has one.
func firstOdo(ts []Trip) float64 {
	for _, t := range ts {
		if t.OdoKm > 0 {
			return t.OdoKm
		}
	}
	return 0
}

// WithEstimates adds, to the trophies that count distance or trips, about when they will be earned at the pace of the last 90 days of
// driving (or of all of it, when it is shorter, once there is a week of it). The odometer trophies go by the odometer's own reading.
func WithEstimates(bs []Badge, ts []Trip, now time.Time) []Badge {
	var latestOdo float64
	cutoff := now.AddDate(0, 0, -90).Unix()
	var km, trips float64
	first := now.Unix()
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		if t.Start < first {
			first = t.Start
		}
		if t.Start >= cutoff {
			km += t.Km
			trips++
		}
		latestOdo = max(latestOdo, t.OdoKm)
	}
	span := now.Sub(time.Unix(max(first, cutoff), 0)).Hours() / 24
	if span < 7 || km <= 0 {
		return bs
	}
	perDay, tripsPerDay := km/span, trips/span
	out := make([]Badge, len(bs))
	copy(out, bs)
	for i, b := range out {
		if b.Earned > 0 || b.Goal <= 0 {
			continue
		}
		var left, rate float64
		switch b.Key {
		case "km160", "km1600", "km16000", "world":
			left, rate = b.Goal-b.Value, perDay
		case "odo50", "odo100", "odo150":
			left, rate = b.Goal-latestOdo, perDay
		case "trips100", "trips1000":
			left, rate = b.Goal-b.Value, tripsPerDay
		default:
			continue
		}
		if left > 0 && rate > 0 && left/rate < 3650 {
			out[i].Eta = now.Add(time.Duration(left / rate * 24 * float64(time.Hour))).Unix()
		}
	}
	return out
}

// MoreBadges works out the extra trophies from the whole history. fills are the fill-ups of a gas or hybrid car.
func MoreBadges(in []Trip, cs []Charge, fills []fuel.Fill, loc *time.Location) []Badge {
	ts := append([]Trip(nil), in...)
	sort.Slice(ts, func(i, j int) bool { return ts[i].Start < ts[j].Start })
	marathon := newBadge("marathon", "Marathon day", "🏃", "Drive 300 miles in one day", "km", 300*kmPerMile)
	night := newBadge("nightrider", "Night rider", "🌃", "Finish a drive between midnight and 4 am", "count", 1)
	lowslow := newBadge("lowslow", "Low and slow", "🐢", "A drive of 50 miles or more at 55 mph or less, at 3.5 mi/kWh or better", "count", 1)
	winter := newBadge("winter", "Winter warrior", "🧊", "A drive of 10 miles or more below 20 °F that beat your average efficiency", "count", 1)
	sunrise := newBadge("sunrise", "Sunrise patrol", "🌅", "Start the day's first drive before 6 am, on 10 days", "count", 10)
	weekend := newBadge("weekend", "Weekend warrior", "🎒", "Drive on 12 weekends in a row", "count", 12)
	regular := newBadge("regular", "Regular", "📍", "Visit the same named place 20 times", "count", 20)
	triple := newBadge("triple", "Triple threat", "🎯", "Arrive at three different named places in one day", "count", 3)
	round := newBadge("round", "Round trip", "🔁", "Leave a named place, drive 100 miles or more that day and come back to it", "count", 1)
	holiday := newBadge("holiday", "Holiday cruiser", "🎄", "Drive on New Year's Day, the Fourth of July, Halloween and Christmas", "count", 4)
	odo50 := newBadge("odo50", "50,000 miles on the clock", "🕰️", "Reach 50,000 miles on the odometer", "km", 50000*kmPerMile)
	odo100 := newBadge("odo100", "100,000 miles on the clock", "🏁", "Reach 100,000 miles on the odometer", "km", 100000*kmPerMile)
	odo150 := newBadge("odo150", "150,000 miles on the clock", "🌠", "Reach 150,000 miles on the odometer", "km", 150000*kmPerMile)

	type day struct {
		km          float64
		tos         map[string]bool
		first, last *Trip
	}
	days := map[string]*day{}
	visits := map[string]int{}
	holidays := map[string]bool{}
	weekendSet := map[int]bool{}
	sunriseDays := map[string]bool{}
	monthKm, monthKwh, monthEnd := map[string]float64{}, map[string]float64{}, map[string]int64{}
	allMonthKm, lastEnd := map[string]float64{}, map[string]int64{}
	var allKm, allKwh float64 // efficiency so far, for the winter trophy
	prior := 0
	for i := range ts {
		t := &ts[i]
		if t.Km < 0.3 {
			continue
		}
		start, end := time.Unix(t.Start, 0).In(loc), time.Unix(t.End, 0).In(loc)
		dk := start.Format("2006-01-02")
		d := days[dk]
		if d == nil {
			d = &day{tos: map[string]bool{}, first: t}
			days[dk] = d
			if start.Hour() < 6 {
				sunriseDays[dk] = true
				sunrise.see(float64(len(sunriseDays)), t.End)
			}
		}
		d.last = t
		d.km += t.Km
		marathon.see(d.km, t.End)
		if t.To != "" {
			d.tos[t.To] = true
			triple.see(float64(len(d.tos)), t.End)
			if t.To != "Home" {
				visits[t.To]++
				regular.see(float64(visits[t.To]), t.End)
			}
		}
		if d.first.From != "" && d.last.To == d.first.From && d.km >= 100*kmPerMile {
			round.see(1, t.End)
		}
		if end.Hour() < 4 {
			night.see(1, t.End)
		}
		if md := start.Format("01-02"); md == "01-01" || md == "07-04" || md == "10-31" || md == "12-25" {
			holidays[md] = true
			holiday.see(float64(len(holidays)), t.End)
		}
		if wd := start.Weekday(); wd == time.Saturday || wd == time.Sunday {
			monday := time.Date(start.Year(), start.Month(), start.Day(), 12, 0, 0, 0, loc).AddDate(0, 0, -((int(wd) + 6) % 7))
			weekendSet[int(monday.Unix()/(7*86400))] = true
			weeks := make([]int, 0, len(weekendSet))
			for w := range weekendSet {
				weeks = append(weeks, w)
			}
			sort.Ints(weeks)
			weekend.see(float64(runOf(weeks)), t.End)
		}
		if t.OdoKm > 0 {
			for _, b := range []*tracker{odo50, odo100, odo150} {
				b.see(t.OdoKm, t.End)
			}
		}
		mk := start.Format("2006-01")
		allMonthKm[mk] += t.Km
		lastEnd[mk] = t.End
		if t.Kwh > 0 && !t.Corrected {
			monthKm[mk] += t.Km
			monthKwh[mk] += t.Kwh
			monthEnd[mk] = t.End
			eff := t.Km / t.Kwh
			hours := float64(t.End-t.Start) / 3600
			if t.Km >= 50*kmPerMile && hours > 0 && t.Km/hours <= 55*kmPerMile && eff >= 3.5*kmPerMile {
				lowslow.see(1, t.End)
			}
			if t.TempC != nil && *t.TempC < -6.7 && t.Km >= 10*kmPerMile && prior >= 10 && eff > allKm/allKwh {
				winter.see(1, t.End)
			}
			allKm += t.Km
			allKwh += t.Kwh
			prior++
		}
	}
	// a milestone the odometer was already past when CarTrak began: reached, but not on the day of the first trip
	if first := firstOdo(ts); first > 0 {
		for _, b := range []*tracker{odo50, odo100, odo150} {
			if first >= b.b.Goal {
				b.b.Note = "Already reached when CarTrak began"
			}
		}
	}
	// the next rungs of ladders that already have a first rung (see js/pages/trophies.js)
	streak7 := newBadge("streak7", "7-day streak", "📆", "Drive 7 days in a row", "days", 7)
	streak100 := newBadge("streak100", "100-day streak", "🔥", "Drive 100 days in a row", "days", 100)
	trips5000 := newBadge("trips5000", "5,000 trips", "🏅", "Make 5,000 trips", "count", 5000)
	regular50 := newBadge("regular50", "Regular, 50 visits", "📌", "Visit the same named place 50 times", "count", 50)
	regular100 := newBadge("regular100", "Regular, 100 visits", "📍", "Visit the same named place 100 times", "count", 100)
	var tripN float64
	runVisits := map[string]int{}
	var dayInts []int
	seenDays := map[int]bool{}
	for i := range ts {
		t := &ts[i]
		if t.Km < 0.3 {
			continue
		}
		tripN++
		trips5000.see(tripN, t.End)
		if t.To != "" && t.To != "Home" {
			runVisits[t.To]++
			regular50.see(float64(runVisits[t.To]), t.End)
			regular100.see(float64(runVisits[t.To]), t.End)
		}
		d := time.Unix(t.Start, 0).In(loc)
		n := int(time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc).Unix() / 86400)
		if !seenDays[n] {
			seenDays[n] = true
			dayInts = append(dayInts, n)
			sort.Ints(dayInts)
			run := float64(runOf(dayInts))
			streak7.see(run, t.End)
			streak100.see(run, t.End)
		}
	}
	out := []*tracker{streak7, streak100, trips5000, regular50, regular100, marathon, night, lowslow, winter, sunrise, weekend, regular, triple, round, holiday, odo50, odo100, odo150}

	// a month more efficient than the months before it, and a busy month with no public charging
	gentle := newBadge("gentle", "Gentle month", "🌿", "Drive a month 10% more efficiently than the months before it", "count", 1)
	skipper := newBadge("skipper", "Gas station skipper", "🏠", "Drive 300 miles in a month without one public charge", "count", 1)
	months := make([]string, 0, len(monthKm))
	for m := range monthKm {
		months = append(months, m)
	}
	sort.Strings(months)
	thisMonth := time.Now().In(loc).Format("2006-01")
	var pKm, pKwh float64
	for i, m := range months {
		if i >= 2 && m != thisMonth && monthKm[m] >= 200 && monthKm[m]/monthKwh[m] >= 1.1*pKm/pKwh {
			gentle.see(1, monthEnd[m])
		}
		pKm += monthKm[m]
		pKwh += monthKwh[m]
	}
	public := map[string]bool{}
	for _, c := range cs {
		if c.Where != "home" && c.Kwh > 0 {
			public[time.Unix(c.Start, 0).In(loc).Format("2006-01")] = true
		}
	}
	for m, km := range allMonthKm {
		if m != thisMonth && km >= 300*kmPerMile && !public[m] && len(cs) > 0 {
			skipper.see(1, lastEnd[m])
		}
	}
	out = append(out, gentle, skipper)

	// charging
	quick := newBadge("quick", "Quick charge", "⏱️", "Add 40 kWh or more in under half an hour", "count", 1)
	detective := newBadge("detective", "Loss detective", "🔍", "Enter the charger's own kWh on 10 sessions, so CarTrak can measure charging loss", "count", 10)
	plug := newBadge("plugstreak", "Plugged-in streak", "🔌", "Start a charge in the evening or night on 14 days in a row", "count", 14)
	var measured float64
	var plugDays []int
	seenDay := map[int]bool{}
	sorted := append([]Charge(nil), cs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	for _, c := range sorted {
		if c.Kwh >= 40 && c.End > c.Start && c.End-c.Start <= 1800 {
			quick.see(1, c.End)
		}
		if c.Eff != nil {
			measured++
			detective.see(measured, c.End)
		}
		s := time.Unix(c.Start, 0).In(loc)
		if s.Hour() >= 17 || s.Hour() < 5 {
			d := time.Date(s.Year(), s.Month(), s.Day(), 12, 0, 0, 0, loc)
			if s.Hour() < 5 {
				d = d.AddDate(0, 0, -1) // after midnight counts for the evening before
			}
			if n := int(d.Unix() / 86400); !seenDay[n] {
				seenDay[n] = true
				plugDays = append(plugDays, n)
				sort.Ints(plugDays)
				plug.see(float64(runOf(plugDays)), c.End)
			}
		}
	}
	out = append(out, quick, detective, plug)
	mwh100 := newBadge("mwh100", "100 MWh", "🏭", "Charge 100,000 kWh in all", "kwh", 100000)
	var kwhTotal float64
	for _, c := range sorted {
		kwhTotal += c.Kwh
		mwh100.see(kwhTotal, c.End)
	}
	out = append(out, mwh100)

	// fuel
	muncher := newBadge("muncher", "Mile muncher", "⛽", "Drive 400 miles between two fill-ups", "km", 400*kmPerMile)
	thrifty := newBadge("thrifty", "Thrifty fill", "💰", "Fill up at a price 10% under your average, once you have 5 fill-ups priced", "count", 1)
	var paid, gal float64
	nPriced := 0
	for i, f := range fills {
		if i > 0 {
			if km, _, _ := fuel.Between(fills[i-1], f, 0); km > 0 {
				muncher.see(km, f.TS)
			}
		}
		if f.Gallons != nil && f.PricePerGal != nil && *f.Gallons > 0 {
			if nPriced >= 5 && *f.PricePerGal <= 0.9*paid/gal {
				thrifty.see(1, f.TS)
			}
			paid += *f.Gallons * *f.PricePerGal
			gal += *f.Gallons
			nPriced++
		}
	}
	out = append(out, muncher, thrifty)

	res := make([]Badge, 0, len(out))
	for _, t := range out {
		res = append(res, t.b)
	}
	return res
}
