package alerts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	kvTireDay    = "alert_tire_day"
	kvTireWinter = "alert_tire_winter" // the year a winter reminder was sent
	kvTireSpring = "alert_tire_spring"
	coldHighC    = 7.0  // 45 F: below this all day, winter tires start to make sense
	coldDays     = 3    // days in a row
	mildHighC    = 10.0 // 50 F: above this, winter tires are done
	mildDays     = 5
)

func f2c(c float64) string { return fmt.Sprintf("%.0f°F (%.0f°C)", c*9/5+32, c) }

// checkTires suggests winter tires when the daily high has stayed under 45 F for three days (autumn and early winter), and suggesting
// the summer set back when it has stayed over 50 F for five (spring), each once a year and only if the tire log says which set is on.
// It reads the car's own outside temperature, so it does not need the weather lookup, and looks at most once a day.
func (e *Engine) checkTires() {
	now := e.now()
	today := now.Unix() / 86400
	if e.S.GetKV(kvTireDay) == strconv.FormatInt(today, 10) {
		return
	}
	_ = e.S.SetKV(kvTireDay, strconv.FormatInt(today, 10))
	sets, err := e.S.TireSets()
	if err != nil || len(sets) == 0 {
		return // nobody has logged tires: no opinion on which set is on
	}
	cur := store.TireAt(sets, now.Unix())
	days, highs, err := e.S.DailyHighs(now.AddDate(0, 0, -10).Unix())
	if err != nil {
		return
	}
	// whole days only: today is still in progress
	var last []float64
	for i, d := range days {
		if d < today && i < len(highs) {
			last = append(last, highs[i])
		}
	}
	allBelow := func(n int, c float64) bool {
		if len(last) < n {
			return false
		}
		for _, h := range last[len(last)-n:] {
			if h >= c {
				return false
			}
		}
		return true
	}
	allAbove := func(n int, c float64) bool {
		if len(last) < n {
			return false
		}
		for _, h := range last[len(last)-n:] {
			if h <= c {
				return false
			}
		}
		return true
	}
	year := strconv.Itoa(now.Year())
	switch m := now.Month(); {
	case (m >= time.October || m <= time.January) && (cur == nil || cur.Kind != "winter") && allBelow(coldDays, coldHighC) && e.S.GetKV(kvTireWinter) != year:
		_ = e.S.SetKV(kvTireWinter, year)
		e.send(push.Alert{Kind: push.TireSwap, Title: "Time for winter tires?", Body: fmt.Sprintf("The daily high has stayed under %s for %d days. Log the swap under More, Tires, and CarTrak will show what it does to your range.", f2c(coldHighC), coldDays), URL: "#/tires"})
	case (m >= time.March && m <= time.May) && cur != nil && cur.Kind == "winter" && allAbove(mildDays, mildHighC) && e.S.GetKV(kvTireSpring) != year:
		_ = e.S.SetKV(kvTireSpring, year)
		e.send(push.Alert{Kind: push.TireSwap, Title: "Time to take the winter tires off?", Body: fmt.Sprintf("The daily high has stayed over %s for %d days. Log the swap under More, Tires.", f2c(mildHighC), mildDays), URL: "#/tires"})
	}
}
