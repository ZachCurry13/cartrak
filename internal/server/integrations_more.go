package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/calendar"
	"github.com/ZachCurry13/cartrak/internal/nws"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

// calendarCard is the Integrations card for the calendar link.
func (s *Server) calendarCard() integration {
	on := s.Cfg.CalendarURL() != ""
	note := "Paste the calendar's private ICS address."
	if on {
		note = "Address saved."
		s.cal.mu.Lock()
		if !s.cal.at.IsZero() {
			if s.cal.err != "" {
				note = "The last check failed: " + s.cal.err
			} else {
				note = fmt.Sprintf("%d appointments read, checked %s.", len(s.cal.events), agoText(s.cal.at.Unix()))
			}
		}
		s.cal.mu.Unlock()
	}
	return integration{ID: "calendar", Name: "Calendar (ICS link)", Purpose: "Today's appointments on Home and in the route planner, trips named after the appointment they went to, and the evening heads-up about tomorrow.",
		Sends: "Nothing about you: CarTrak only downloads the calendar. Appointment places are looked up like an address search.",
		State: onOff(on), Note: note, Keys: []string{settings.CalendarURLKey, settings.CalendarWorkKey}, Testable: on}
}

// testMore tests the services added after the first Integrations page. handled is false for an unknown id.
func (s *Server) testMore(ctx context.Context, id string) (handled bool, err error) {
	switch id {
	case "calendar":
		if s.Cfg.CalendarURL() == "" {
			return true, fmt.Errorf("add the address first")
		}
		text, err := calendar.Fetch(ctx, nil, s.Cfg.CalendarURL())
		if err != nil {
			return true, err
		}
		n := len(calendar.Between(calendar.Parse(text, s.Cfg.Location()), time.Now(), time.Now().AddDate(0, 0, 7)))
		s.cal.mu.Lock()
		s.cal.at = time.Time{} // read it again on the next look
		s.cal.mu.Unlock()
		if n == 0 {
			return true, fmt.Errorf("the calendar answered, but has no appointments in the next week")
		}
		return true, nil
	case "nws":
		return true, nws.Ping(ctx, nil, s.NWSURL)
	}
	return false, nil
}
