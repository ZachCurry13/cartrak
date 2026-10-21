package analytics

import (
	"fmt"
	"sort"
)

// CoachCharge is a finished session with what the coach needs.
type CoachCharge struct {
	Start, End   int64
	Kwh, WallKwh float64
	Cost         *float64
	Where, Place string
	PeakKw       float64
	StartSoC     *float64
	EndSoC       *float64
	CheapShare   float64 // 0 to 1: the part of the session inside the cheaper night hours (0 when there are none)
	NextUsePct   *float64
}

// Prices are what a kWh costs, for the money tips. Cheap is the night rate, 0 when there is none.
type Prices struct {
	Home, Cheap, Public float64
	Currency            string
}

// Tip is one piece of advice: a short title, the numbers behind it, and how much it is worth when that can be said.
type Tip struct {
	Key   string  `json:"key"`
	Emoji string  `json:"emoji"`
	Title string  `json:"title"`
	Body  string  `json:"body"`
	Worth float64 `json:"worth,omitempty"` // money a month, when known
}

// Coach looks at about two months of charging and gives the tips the numbers clearly support, the most valuable first.
func Coach(cs []CoachCharge, p Prices, days float64) []Tip {
	tips := []Tip{}
	cur := p.Currency
	if cur == "" {
		cur = "$"
	}
	month := 30 / max(days, 1)
	// 1. charging to full for everyday driving
	var home, full, small int
	for _, c := range cs {
		if c.Where != "home" || c.EndSoC == nil {
			continue
		}
		home++
		if *c.EndSoC >= 95 {
			full++
			if c.NextUsePct != nil && *c.NextUsePct < 40 {
				small++
			}
		}
	}
	if home >= 5 && float64(full) >= 0.6*float64(home) && small*2 >= full {
		tips = append(tips, Tip{Key: "full", Emoji: "🔋", Title: "Charging to 100% most nights",
			Body: fmt.Sprintf("%d of your last %d home charges went to 95%% or more, but the driving after them usually used less than 40%%. For everyday driving, a target of 80 to 90%% is easier on the battery; charge to 100%% before a long trip.", full, home)})
	}
	// 2. the night rate
	if p.Cheap > 0 && p.Home > p.Cheap {
		var dayKwh float64
		for _, c := range cs {
			if c.Where == "home" {
				wall := max(c.WallKwh, c.Kwh)
				dayKwh += wall * (1 - c.CheapShare)
			}
		}
		if save := dayKwh * (p.Home - p.Cheap) * month; save >= 3 {
			tips = append(tips, Tip{Key: "night", Emoji: "🌙", Title: fmt.Sprintf("About %s%.0f a month by charging at night", cur, save),
				Body: fmt.Sprintf("%.0f kWh of your home charging was outside the cheaper night hours. Starting those charges in the night window (a charge schedule in the car or the charger) would have cost about %s%.0f a month less.", dayKwh, cur, save), Worth: save})
		}
	}
	// 3. a fast charger that is slow for you
	var fast []float64
	byPlace := map[string][]float64{}
	for _, c := range cs {
		if c.PeakKw < 30 || c.End <= c.Start || c.Kwh < 5 {
			continue
		}
		avg := c.Kwh / (float64(c.End-c.Start) / 3600)
		fast = append(fast, avg)
		if c.Place != "" {
			byPlace[c.Place] = append(byPlace[c.Place], avg)
		}
	}
	if len(fast) >= 4 {
		usual := medianF(fast)
		for place, v := range byPlace {
			if m := medianF(v); len(v) >= 2 && m < usual*0.65 {
				tips = append(tips, Tip{Key: "slow", Emoji: "🐢", Title: fmt.Sprintf("%s is slow for you", place),
					Body: fmt.Sprintf("Your %d fast charges there averaged %.0f kW, against %.0f kW at your other fast chargers. Another stall, or another charger nearby, may save time.", len(v), m, usual)})
			}
		}
	}
	// 4. arriving very low
	var low int
	for _, c := range cs {
		if c.StartSoC != nil && *c.StartSoC < 10 {
			low++
		}
	}
	if low >= 3 {
		tips = append(tips, Tip{Key: "low", Emoji: "⚠️", Title: fmt.Sprintf("Under 10%% at the charger %d times", low),
			Body: "Arriving that low leaves little room if a charger is busy or broken. The road-trip planner keeps your battery reserve in hand; stopping a little earlier also charges faster."})
	}
	// 5. what public charging costs on top
	if p.Home > 0 {
		var pubKwh, pubCost float64
		for _, c := range cs {
			if c.Where == "public" && c.Cost != nil && c.Kwh > 0 {
				pubKwh += max(c.WallKwh, c.Kwh)
				pubCost += *c.Cost
			}
		}
		if extra := (pubCost - pubKwh*p.Home) * month; pubKwh >= 30 && pubCost/pubKwh >= 2*p.Home && extra >= 10 {
			tips = append(tips, Tip{Key: "public", Emoji: "🏠", Title: fmt.Sprintf("Public charging costs about %s%.0f a month more", cur, extra),
				Body: fmt.Sprintf("You paid about %s%.2f a kWh in public against %s%.2f at home. Every charge you can move home saves the difference.", cur, pubCost/pubKwh, cur, p.Home), Worth: extra})
		}
	}
	sort.SliceStable(tips, func(i, j int) bool { return tips[i].Worth > tips[j].Worth })
	return tips
}

func medianF(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}
