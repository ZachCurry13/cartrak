package alerts

import (
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	kvDigestWeek = "alert_digest_week"
	digestHour   = 18            // Sunday evening, in the car's time zone
	digestWindow = 6 * time.Hour // it may go out up to this long after the hour
)

// weeklyDigest sends one summary of the last seven days on Sunday evening: distance, trips, energy added by
// charging and efficiency. It is sent only to devices that turned on "Weekly summary".
func (e *Engine) weeklyDigest() {
	loc := time.UTC
	if e.Loc != nil {
		loc = e.Loc()
	}
	now := e.now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), digestHour, 0, 0, 0, loc)
	if now.Weekday() != time.Sunday || now.Before(start) || now.After(start.Add(digestWindow)) {
		return
	}
	year, week := now.ISOWeek()
	key := fmt.Sprintf("%d-%d", year, week)
	if e.S.GetKV(kvDigestWeek) == key {
		return
	}
	since := now.AddDate(0, 0, -7).Unix()
	ts, err := e.S.EnergyTrips(since)
	if err != nil {
		return
	}
	var km, kwh float64
	n := 0
	for _, t := range ts {
		n++
		km += t.DistanceKm
		if en, ok := trips.EnergyUsed(t, nil); ok && !en.Estimated {
			kwh += en.Kwh
		}
	}
	if n == 0 {
		_ = e.S.SetKV(kvDigestWeek, key) // nothing to report this week
		return
	}
	var added float64
	sessions := 0
	if cs, err := e.S.ListCharges(100); err == nil {
		for _, c := range cs {
			if c.StartTS >= since && !c.Open && c.StartKwh != nil && c.EndKwh != nil && *c.EndKwh > *c.StartKwh {
				added += *c.EndKwh - *c.StartKwh
				sessions++
			}
		}
	}
	body := fmt.Sprintf("%.0f mi (%.0f km) in %d trip%s.", km*0.621371, km, n, plural(n))
	if sessions > 0 {
		body += fmt.Sprintf(" %d charge%s added %.0f kWh.", sessions, plural(sessions), added)
	}
	if kwh > 0 {
		body += fmt.Sprintf(" %.1f mi/kWh.", km*0.621371/kwh)
	}
	_ = e.S.SetKV(kvDigestWeek, key)
	e.send(push.Alert{Kind: push.WeeklySummary, Title: "Your week with the car", Body: body, URL: "#/insights"})
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
