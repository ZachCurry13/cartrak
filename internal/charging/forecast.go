package charging

// Forecast is what adding energy to the battery would cost.
type Forecast struct {
	BatteryKwh  float64  `json:"batteryKwh"` // goes into the battery
	HomeKwh     float64  `json:"homeKwh"`    // taken from the wall on a slow charger, including heat loss
	HomeCost    *float64 `json:"homeCost"`
	PublicKwh   float64  `json:"publicKwh"` // fast chargers: no loss counted
	PublicCost  *float64 `json:"publicCost"`
	OffPeakCost *float64 `json:"offPeakCost"` // at home inside the cheaper hours, when the plan has them
}

// ForecastCharge prices going from one battery percentage to another. acEff is the share of wall energy that
// reaches the battery on slow charging. It returns false when the range makes no sense or the capacity is unknown.
func ForecastCharge(capacityKwh, fromPct, toPct float64, p Prices, acEff float64) (Forecast, bool) {
	if capacityKwh <= 0 || fromPct < 0 || toPct > 100 || toPct <= fromPct {
		return Forecast{}, false
	}
	if acEff < 0.5 || acEff > 1 {
		acEff = acEfficiency
	}
	f := Forecast{BatteryKwh: (toPct - fromPct) / 100 * capacityKwh}
	f.HomeKwh = f.BatteryKwh / acEff
	f.PublicKwh = f.BatteryKwh
	if p.Home != nil {
		c := f.HomeKwh * *p.Home
		f.HomeCost = &c
	}
	if p.OffPeak != nil {
		c := f.HomeKwh * *p.OffPeak
		f.OffPeakCost = &c
	}
	if p.Public != nil {
		c := f.PublicKwh * *p.Public
		f.PublicCost = &c
	}
	return f, true
}

// MonthProjection estimates the month's charging bill from the part of it that has passed. dayOfMonth
// is the days elapsed so far (fractional). With under three days of data it says no,
// since one session would swing it wildly.
func MonthProjection(spent, dayOfMonth float64, daysInMonth int) (float64, bool) {
	if spent <= 0 || dayOfMonth < 3 || daysInMonth <= 0 {
		return 0, false
	}
	if dayOfMonth >= float64(daysInMonth) {
		return spent, true
	}
	return spent / dayOfMonth * float64(daysInMonth), true
}
