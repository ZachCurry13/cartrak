// Package planahead forecasts the next week of driving from the last weeks of it: how far, how much energy, and when the battery
// would need a charge if you did not plug in.
package planahead

import (
	"time"
)

const (
	// MinHistoryDays is how much driving history is needed before a forecast is offered.
	MinHistoryDays = 14
	// Horizon is how many days ahead the forecast looks.
	Horizon = 7
)

// DayKm is the distance driven on one local calendar day.
type DayKm struct {
	Date string // 2006-01-02
	Km   float64
}

// Input is everything the forecast needs.
type Input struct {
	History     []DayKm // completed days, oldest first, zero days included; today is not in it
	Today       time.Time
	KwhPerKm    float64 // recent energy use per km
	CapacityKwh float64
	StartPct    float64 // battery level now
	ReservePct  float64 // the level to stay above
	// Away is set on a trip away: every day is the average of the trip so far (your weekdays at home do not fit a trip), the
	// forecast runs to the trip's last day (at most Horizon), and two days of the trip are enough.
	Away     bool
	DaysLeft int // on a trip away, the days still to come, from tomorrow
}

// Day is one forecast day.
type Day struct {
	Date    string  `json:"date"`
	Weekday string  `json:"weekday"`
	Km      float64 `json:"km"`
	Kwh     float64 `json:"kwh"`
	EndPct  float64 `json:"endPct"` // the battery level at the end of the day with no charging
}

// Forecast is the week ahead.
type Forecast struct {
	Days        []Day   `json:"days"`
	TotalKm     float64 `json:"totalKm"`
	TotalKwh    float64 `json:"totalKwh"`
	StartPct    float64 `json:"startPct"`
	EndPct      float64 `json:"endPct"`
	ReservePct  float64 `json:"reservePct"`
	ChargeBy    string  `json:"chargeBy"` // the date of the first day that would end below the reserve, "" when none does
	ChargeByDay string  `json:"chargeByDay"`
	HistoryDays int     `json:"historyDays"`
}

// Build forecasts the next Horizon days after Today. Each day is the average of the same weekday in the history; a weekday seen
// fewer than twice uses the average of all days. It reports false when there is too little history or no energy figure.
func Build(in Input) (Forecast, bool) {
	need, horizon := MinHistoryDays, Horizon
	if in.Away {
		need, horizon = 2, min(Horizon, in.DaysLeft)
	}
	if len(in.History) < need || horizon < 1 || in.KwhPerKm <= 0 || in.CapacityKwh <= 0 {
		return Forecast{}, false
	}
	var sum [7]float64
	var n [7]int
	var all float64
	for _, h := range in.History {
		t, err := time.Parse("2006-01-02", h.Date)
		if err != nil {
			continue
		}
		sum[t.Weekday()] += h.Km
		n[t.Weekday()]++
		all += h.Km
	}
	mean := all / float64(len(in.History))
	f := Forecast{StartPct: in.StartPct, ReservePct: in.ReservePct, HistoryDays: len(in.History)}
	pct := in.StartPct
	for i := 1; i <= horizon; i++ {
		d := in.Today.AddDate(0, 0, i)
		km := mean
		if !in.Away && n[d.Weekday()] >= 2 {
			km = sum[d.Weekday()] / float64(n[d.Weekday()])
		}
		kwh := km * in.KwhPerKm
		pct -= kwh / in.CapacityKwh * 100
		day := Day{Date: d.Format("2006-01-02"), Weekday: d.Weekday().String(), Km: km, Kwh: kwh, EndPct: pct}
		f.Days = append(f.Days, day)
		f.TotalKm += km
		f.TotalKwh += kwh
		if f.ChargeBy == "" && pct < in.ReservePct {
			f.ChargeBy, f.ChargeByDay = day.Date, day.Weekday
		}
	}
	f.EndPct = pct
	return f, true
}
