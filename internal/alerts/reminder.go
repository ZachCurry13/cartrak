package alerts

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/maint"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	kvPlugDay      = "alert_plug_reminder_day"
	reminderWindow = 90 * time.Minute // how long after the set time a reminder may still go out
)

// plugReminder nudges once an evening when the car is parked, not plugged in and below the chosen level.
// It looks at the latest reading, so it stays quiet if the car has not reported for a couple of hours.
func (e *Engine) plugReminder() {
	if e.ReminderTime == nil || e.ReminderBelow == nil {
		return
	}
	at := e.ReminderTime()
	if at == "" {
		return
	}
	clock, err := time.Parse("15:04", at)
	if err != nil {
		return
	}
	loc := time.UTC
	if e.Loc != nil {
		loc = e.Loc()
	}
	now := e.now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
	if now.Before(start) || now.After(start.Add(reminderWindow)) {
		return
	}
	day := now.Format("2006-01-02")
	if e.S.GetKV(kvPlugDay) == day {
		return
	}
	st := e.prev
	if st == nil || e.now().Sub(st.TS) > 2*time.Hour || st.SoCPct == nil || *st.SoCPct >= e.ReminderBelow() {
		return
	}
	if p, known := plugged(st); !known || p || charging(st) || trips.Driving(st) {
		return
	}
	_ = e.S.SetKV(kvPlugDay, day)
	e.send(push.Alert{Kind: push.PlugReminder, Title: "Plug in tonight?",
		Body: fmt.Sprintf("The car is at %.0f%% and not plugged in.", *st.SoCPct), URL: "#/charging"})
}

const (
	interruptedBelowPct = 70.0 // a session that stops lower than this, unlike the ones before it, looks interrupted
	usualStopPct        = 3.0  // within this of where recent sessions ended counts as the usual stopping point
)

// stoppedEarly says a charge that just stopped while still plugged in ended lower than usual. Owners who set
// a low charge limit see their sessions end at the same level every time, so that level is never flagged.
func (e *Engine) stoppedEarly(st *ford.State) bool {
	if st.SoCPct == nil || *st.SoCPct >= interruptedBelowPct {
		return false
	}
	recent, err := e.S.ListCharges(7)
	if err != nil {
		return false
	}
	var ends []float64
	for _, c := range recent {
		// The session that has just ended is not history: it ended where it ended because of what is being judged.
		if !c.Open && c.EndSoC != nil && (c.EndTS == nil || *c.EndTS < st.TS.Unix()-900) {
			ends = append(ends, *c.EndSoC)
		}
	}
	for _, v := range ends {
		if v-*st.SoCPct < usualStopPct && *st.SoCPct-v < usualStopPct {
			return false // it stopped about where it stopped before
		}
	}
	return true
}

// maintenance sends one alert when a service reminder becomes due by distance or by time. Marking the work
// done re-arms it.
func (e *Engine) maintenance(odoKm *float64) {
	items, err := e.S.ListMaintenance()
	if err != nil {
		return
	}
	for _, m := range items {
		if m.NotifiedTS != 0 {
			continue
		}
		st := maint.Check(m, odoKm, e.now().Unix())
		if st.State != maint.Overdue {
			continue
		}
		_ = e.S.SetMaintenanceNotified(m.ID)
		e.send(push.Alert{Kind: push.MaintenanceDue, Title: "Service due: " + m.Name, Body: "This reminder is due. Open Maintenance to see it, and mark it done when it is finished.", URL: "#/maintenance"})
	}
}

const leaveMargin = 1.25 // the car must get this much farther than the place's size before it counts as left

// namedPlaces announces arriving at and leaving the places the owner named. The first reading after a restart
// only sets the starting point, and leaving needs a margin so a wobbling position at the edge does not repeat.
func (e *Engine) namedPlaces(st *ford.State, first bool) {
	if st.Lat == nil || st.Lon == nil {
		return
	}
	all, err := e.S.ListNamedPlaces()
	if err != nil {
		return
	}
	if e.inside == nil {
		e.inside = map[int64]bool{}
	}
	// The car often repeats its last position for a whole drive, so how far the odometer has gone since the position last changed
	// says it has left a place long before a new position arrives.
	if !e.havePos || e.posLat != *st.Lat || e.posLon != *st.Lon {
		e.posLat, e.posLon, e.havePos = *st.Lat, *st.Lon, true
		e.posOdo = -1
		if st.OdometerKm != nil {
			e.posOdo = *st.OdometerKm
		}
	}
	drivenM := 0.0
	if e.posOdo >= 0 && st.OdometerKm != nil {
		drivenM = (*st.OdometerKm - e.posOdo) * 1000
	}
	seen := map[int64]bool{}
	for _, p := range all {
		seen[p.ID] = true
		d := places.Distance(*st.Lat, *st.Lon, p)
		was := e.inside[p.ID]
		now := was
		switch {
		case !was && d <= p.RadiusM && drivenM <= p.RadiusM*leaveMargin+500: // a repeated old position is not an arrival
			now = true
		case was && (d > p.RadiusM*leaveMargin || drivenM > p.RadiusM*leaveMargin+500):
			now = false
		}
		e.inside[p.ID] = now
		if first || now == was {
			continue
		}
		if now {
			e.send(push.Alert{Kind: push.ArrivePlace, Title: "Arrived at " + p.Name, Body: "The car is at " + p.Name + "." + charge(st), URL: "#/"})
		} else {
			e.send(push.Alert{Kind: push.LeavePlace, Title: "Left " + p.Name, Body: "The car has left " + p.Name + "." + charge(st), URL: "#/"})
		}
	}
	for id := range e.inside {
		if !seen[id] {
			delete(e.inside, id)
		}
	}
}

// chargeFault says in words why the car reports the charge ended badly, or "" when its charge state is not a fault.
func chargeFault(st *ford.State) string {
	switch strings.ToUpper(st.ChargeState) {
	case "FAULT":
		return "The charger reported a fault. "
	case "STATION_NOT_DETECTED":
		return "The car lost contact with the charger. "
	}
	return ""
}
