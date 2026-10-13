// Package maint works out whether a service reminder is due, from the odometer and the calendar.
package maint

import (
	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	soonKm   = 500.0 // about 300 miles
	soonDays = 30
)

// State is how close a reminder is.
type State string

const (
	OK      State = "ok"
	Soon    State = "soon"
	Overdue State = "overdue"
)

// Status is one reminder with how much is left of it. Left values are negative once overdue and nil when the
// reminder does not use that kind of interval (or the odometer is not known yet).
type Status struct {
	*store.Maintenance
	State    State    `json:"state"`
	KmLeft   *float64 `json:"kmLeft"`
	DaysLeft *int     `json:"daysLeft"`
}

// Check works out the state of m at odometer odoKm (nil if unknown) and time now (unix seconds).
func Check(m *store.Maintenance, odoKm *float64, now int64) Status {
	st := Status{Maintenance: m, State: OK}
	if m.EveryKm != nil && m.LastOdoKm != nil && odoKm != nil {
		left := *m.LastOdoKm + *m.EveryKm - *odoKm
		st.KmLeft = &left
	}
	if m.EveryDays != nil && m.LastTS > 0 {
		left := int((m.LastTS + int64(*m.EveryDays)*86400 - now) / 86400)
		if m.LastTS+int64(*m.EveryDays)*86400 < now && left == 0 {
			left = -1 // due earlier today counts as overdue
		}
		st.DaysLeft = &left
	}
	switch {
	case (st.KmLeft != nil && *st.KmLeft <= 0) || (st.DaysLeft != nil && *st.DaysLeft < 0):
		st.State = Overdue
	case (st.KmLeft != nil && *st.KmLeft <= soonKm) || (st.DaysLeft != nil && *st.DaysLeft <= soonDays):
		st.State = Soon
	}
	return st
}
