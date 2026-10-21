package analytics

import "time"

// Gas is the car CarTrak compares against, and the grid's CO2, for the recap's savings. Zero values leave savings out.
type Gas struct {
	LitresPerKm   float64
	PricePerLitre float64
	GridKgPerKwh  float64
}

const kgCO2PerLitre = 2.31 // burning a litre of gasoline

// Recap is a month or a year told in numbers, made to be shared.
type Recap struct {
	Period     string   `json:"period"` // 2026-10 or 2026
	From       int64    `json:"from"`
	To         int64    `json:"to"`
	Trips      int      `json:"trips"`
	Km         float64  `json:"km"`
	Hours      float64  `json:"hours"`
	DaysDriven int      `json:"daysDriven"`
	KwhUsed    float64  `json:"kwhUsed"`
	KwhCharged float64  `json:"kwhCharged"`
	Sessions   int      `json:"sessions"`
	Cost       float64  `json:"cost"`
	HomeShare  float64  `json:"homeShare"`
	KmPerKwh   float64  `json:"kmPerKwh"`
	Longest    *Trip    `json:"longest,omitempty"`
	BusiestDay *DayKm   `json:"busiestDay,omitempty"`
	PrevKm     *float64 `json:"prevKm,omitempty"` // the period before, to compare
	GasCost    *float64 `json:"gasCost,omitempty"`
	Saved      *float64 `json:"saved,omitempty"`
	KgCO2Saved *float64 `json:"kgCo2Saved,omitempty"`
}

// Period is a calendar month (2026-10) or year (2026) in loc: where it starts and ends, and where the one before starts.
func Period(p string, loc *time.Location) (from, to, prevFrom int64, ok bool) {
	if t, err := time.ParseInLocation("2006-01", p, loc); err == nil {
		return t.Unix(), t.AddDate(0, 1, 0).Unix(), t.AddDate(0, -1, 0).Unix(), true
	}
	if t, err := time.ParseInLocation("2006", p, loc); err == nil {
		return t.Unix(), t.AddDate(1, 0, 0).Unix(), t.AddDate(-1, 0, 0).Unix(), true
	}
	return 0, 0, 0, false
}

// PeriodRecap sums up a month or a year (see Period) from all trips and charges.
func PeriodRecap(period string, ts []Trip, cs []Charge, gas Gas, loc *time.Location) Recap {
	r := Recap{Period: period}
	from, to, prevFrom, ok := Period(period, loc)
	if !ok {
		return r
	}
	r.From, r.To = from, to
	var prevKm, effKm float64
	prevSeen := false
	days := map[string]*DayKm{}
	for _, t := range ts {
		if t.Start >= prevFrom && t.Start < from && t.Km >= 0.3 {
			prevKm += t.Km
			prevSeen = true
		}
		if t.Start < from || t.Start >= to || t.Km < 0.3 {
			continue
		}
		r.Trips++
		r.Km += t.Km
		r.Hours += float64(t.End-t.Start) / 3600
		if t.Kwh > 0 {
			r.KwhUsed += t.Kwh
			effKm += t.Km
		}
		if r.Longest == nil || t.Km > r.Longest.Km {
			tt := t
			r.Longest = &tt
		}
		d := time.Unix(t.Start, 0).In(loc).Format("2006-01-02")
		if days[d] == nil {
			days[d] = &DayKm{Date: d}
		}
		days[d].Km += t.Km
		days[d].Trips++
	}
	r.DaysDriven = len(days)
	for _, d := range days {
		if r.BusiestDay == nil || d.Km > r.BusiestDay.Km {
			r.BusiestDay = d
		}
	}
	if prevSeen {
		r.PrevKm = &prevKm
	}
	if r.KwhUsed > 0 {
		r.KmPerKwh = effKm / r.KwhUsed
	}
	var home float64
	for _, c := range cs {
		if c.Start < from || c.Start >= to || c.Kwh <= 0 {
			continue
		}
		r.Sessions++
		r.KwhCharged += c.Kwh
		if c.Where == "home" {
			home += c.Kwh
		}
		if c.Cost != nil {
			r.Cost += *c.Cost
		}
	}
	if r.KwhCharged > 0 {
		r.HomeShare = home / r.KwhCharged
	}
	if gas.LitresPerKm > 0 && gas.PricePerLitre > 0 && r.Km > 0 {
		litres := r.Km * gas.LitresPerKm
		g := litres * gas.PricePerLitre
		r.GasCost = &g
		if r.Cost > 0 && r.KmPerKwh > 0 {
			saved := g - r.Cost/r.KwhCharged*(r.Km/r.KmPerKwh) // the cost of the energy this month's driving used
			r.Saved = &saved
		}
		if gas.GridKgPerKwh >= 0 && r.KmPerKwh > 0 {
			kg := litres*kgCO2PerLitre - r.Km/r.KmPerKwh*gas.GridKgPerKwh
			r.KgCO2Saved = &kg
		}
	}
	return r
}
