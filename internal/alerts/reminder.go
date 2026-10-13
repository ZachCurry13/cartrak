package alerts

import (
	"fmt"
	"time"

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
