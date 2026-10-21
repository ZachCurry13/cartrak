package remotestart

// A remote start as one event: when it began and ended, the battery before and after, and the weather. Readings arrive one
// at a time; Step takes each and says what changed.

// Reading is what one reading says that matters here.
type Reading struct {
	TS      int64
	Left    float64  // countdown, seconds; 0 when none is running
	RunFor  float64  // the car's remote-start run time setting, seconds; 0 when unknown
	SoC     *float64 // battery percent
	Kwh     *float64 // energy left in the battery
	TempC   *float64 // outside
	Driving bool
}

// Session is one remote start. End is 0 while it is running. Restarts counts the times it was started again before it ended
// (the car allows a second run). Drove says it ended because the car was driven off.
type Session struct {
	ID       int64    `json:"id"`
	Start    int64    `json:"start"`
	End      int64    `json:"end"`
	StartSoC *float64 `json:"startSoc"`
	EndSoC   *float64 `json:"endSoc"`
	StartKwh *float64 `json:"startKwh"`
	EndKwh   *float64 `json:"endKwh"`
	TempC    *float64 `json:"tempC"`
	Restarts int      `json:"restarts"`
	Drove    bool     `json:"drove"`
	LastTS   int64    `json:"-"` // the last reading that saw it running, and what was left then
	LastLeft float64  `json:"-"`
}

// UsedKwh is the energy the start took from the battery, when both ends are known.
func (s Session) UsedKwh() *float64 {
	if s.StartKwh == nil || s.EndKwh == nil || *s.StartKwh < *s.EndKwh {
		return nil
	}
	v := *s.StartKwh - *s.EndKwh
	return &v
}

// maxGap is how long a run can go unseen and still be the same one: longer than any run time the car allows.
const maxGap = 45 * 60

// Step applies one reading to the running session (nil when none) and returns the running session afterwards, and the one
// that ended, if any. before is the previous reading, used for the battery at the start.
func Step(open *Session, r Reading, before *Reading) (running, ended *Session) {
	if open != nil && (r.TS-open.LastTS > maxGap || r.Left <= 0 || r.Driving) {
		open.End = min(r.TS, open.LastTS+int64(open.LastLeft))
		if r.Driving {
			open.End, open.Drove = r.TS, true
		}
		if open.End < open.LastTS {
			open.End = open.LastTS
		}
		if r.TS-open.LastTS <= maxGap { // this reading is just after it ended: its battery is the end's
			open.EndSoC, open.EndKwh = r.SoC, r.Kwh
		}
		ended, open = open, nil
	}
	if r.Left <= 0 || r.Driving {
		return nil, ended
	}
	if open == nil {
		start := r.TS
		if r.RunFor > r.Left && r.RunFor-r.Left < maxGap {
			start = r.TS - int64(r.RunFor-r.Left) // it began this long before the countdown was seen
		}
		open = &Session{Start: start, StartSoC: r.SoC, StartKwh: r.Kwh, TempC: r.TempC}
		if before != nil && before.TS >= start-20*60 && before.Left <= 0 { // the reading before shows the battery before it began
			open.StartSoC, open.StartKwh = pick(before.SoC, r.SoC), pick(before.Kwh, r.Kwh)
		}
	} else if r.Left > open.LastLeft+60 {
		open.Restarts++
	}
	open.LastTS, open.LastLeft = r.TS, r.Left
	open.EndSoC, open.EndKwh = r.SoC, r.Kwh
	if open.TempC == nil {
		open.TempC = r.TempC
	}
	return open, ended
}

func pick(a, b *float64) *float64 {
	if a != nil {
		return a
	}
	return b
}
