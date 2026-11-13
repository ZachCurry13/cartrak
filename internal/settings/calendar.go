package settings

import (
	"fmt"
	"strings"
)

// The calendar link, what its appointments count as, and the weather alerts (which use the calendar for the heads-up).
const (
	CalendarURLKey   = "CARTRAK_CALENDAR_URL"
	CalendarWorkKey  = "CARTRAK_CALENDAR_WORK"
	WeatherAlertsKey = "CARTRAK_WEATHER_ALERTS"
)

var (
	calendarWorkOptions  = []Option{{"", "Only name the trip"}, {"work", "Name the trip and count it as Work"}}
	weatherAlertsOptions = []Option{{"", "Off (nothing is sent)"}, {"on", "On: check for warnings near the car and on planned routes"}}
)

func init() {
	Defs = append(Defs,
		Def{Key: CalendarURLKey, Label: "Calendar address (private ICS link)", Secret: true, Placeholder: "https://calendar.google.com/calendar/ical/.../basic.ics",
			Help: "Optional. The private address of a calendar, so CarTrak can list today's appointments, plan a route through them and name the trips that go to them. In Google Calendar: Settings, the calendar, Integrate calendar, Secret address in iCal format. In Outlook: Settings, Calendar, Shared calendars, Publish a calendar, the ICS link. Anyone with the address can read the calendar, so it stays on the server and is never shown again. Appointment places are looked up like an address search."},
		Def{Key: CalendarWorkKey, Label: "Trips to calendar appointments", Options: calendarWorkOptions,
			Help: "A trip that ends at an appointment's place around its time gets the appointment's title. Choose Work to count those trips as work too (you can still change each one)."},
		Def{Key: WeatherAlertsKey, Label: "Weather warnings (National Weather Service)", Options: weatherAlertsOptions,
			Help: "On: CarTrak asks the US National Weather Service (free, no account) for warnings such as winter storms, ice, floods and high wind where the car is, along a planned route and at tomorrow's appointments, and can send them as an alert. Only a rounded area (about 10 km) is sent. US only. Off by default."},
	)
	validators[CalendarWorkKey] = oneOf(calendarWorkOptions)
	validators[WeatherAlertsKey] = oneOf(weatherAlertsOptions)
	validators[CalendarURLKey] = func(v string) error {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "webcal://") {
			return nil
		}
		return fmt.Errorf("paste the address that starts with https:// or webcal://")
	}
}

func oneOf(opts []Option) func(string) error {
	return func(v string) error {
		for _, o := range opts {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the options")
	}
}

// CalendarURL is the private calendar address, "" when not set.
func (s *Settings) CalendarURL() string { return strings.TrimSpace(s.Value(CalendarURLKey)) }

// CalendarWork says whether trips to appointments count as work.
func (s *Settings) CalendarWork() bool { return s.Value(CalendarWorkKey) == "work" }

// WeatherAlerts says whether the National Weather Service warnings are switched on.
func (s *Settings) WeatherAlerts() bool { return s.Value(WeatherAlertsKey) == "on" }
