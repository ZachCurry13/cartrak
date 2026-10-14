package charging

import "time"

// Window is a daily span in minutes after midnight, local time. It may cross midnight (From > To).
type Window struct{ From, To int }

func (w Window) contains(minute int) bool {
	if w.From < w.To {
		return minute >= w.From && minute < w.To
	}
	return minute >= w.From || minute < w.To
}

// Share is the fraction (0 to 1) of the span [startTS, endTS] that falls inside the window in loc. It samples every
// five minutes, which is plenty for a charging session and handles windows that cross midnight and long sessions.
func (w Window) Share(startTS, endTS int64, loc *time.Location) float64 {
	if endTS <= startTS {
		t := time.Unix(startTS, 0).In(loc)
		if w.contains(t.Hour()*60 + t.Minute()) {
			return 1
		}
		return 0
	}
	const step = 300
	in, all := 0, 0
	for ts := startTS; ts < endTS; ts += step {
		t := time.Unix(ts, 0).In(loc)
		all++
		if w.contains(t.Hour()*60 + t.Minute()) {
			in++
		}
	}
	return float64(in) / float64(all)
}
