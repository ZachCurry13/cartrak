package server

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/nws"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

const (
	dayAheadHour  = 19 // the heads-up about tomorrow goes out from 7 pm, once
	kvDayAheadFor = "day_ahead_for"
	roadFactor    = 1.25 // a straight line between stops, made longer to stand for the roads
)

// dayAheadOnce sends the heads-up about tomorrow's appointments once an evening.
func (s *Server) dayAheadOnce(ctx context.Context, now time.Time) {
	loc := s.Cfg.Location()
	local := now.In(loc)
	tomorrow := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	if local.Hour() < dayAheadHour || s.S.GetKV(kvDayAheadFor) == tomorrow.Format("2006-01-02") || s.Cfg.CalendarURL() == "" {
		return
	}
	title, body, ok := s.dayAhead(ctx, tomorrow)
	_ = s.S.SetKV(kvDayAheadFor, tomorrow.Format("2006-01-02"))
	if ok {
		s.tell(ctx, push.Alert{Kind: push.DayAhead, Title: title, Body: body, URL: "#/plan"})
	}
}

// dayAhead is the heads-up for a day: its appointments, the weather and warnings at them, and the battery the day needs.
func (s *Server) dayAhead(ctx context.Context, day time.Time) (string, string, bool) {
	list, _, on := s.appointments(ctx, day)
	var stops []appointment
	for _, a := range list {
		if !a.AllDay {
			stops = append(stops, a)
		}
	}
	if !on || len(stops) == 0 {
		return "", "", false
	}
	us := s.Cfg.LiveGasUnits()
	loc := s.Cfg.Location()
	first := stops[0]
	title := fmt.Sprintf("Tomorrow: %d appointment%s", len(stops), map[bool]string{true: "", false: "s"}[len(stops) == 1])
	parts := []string{fmt.Sprintf("First at %s, %s.", time.Unix(first.Start, 0).In(loc).Format("3:04 pm"), first.Title)}
	var placed []*companyPick
	for _, a := range stops {
		if a.Place != nil {
			placed = append(placed, a.Place)
		}
	}
	lowC := math.NaN()
	if len(placed) > 0 && s.Cfg.WeatherLookup() {
		lowC, parts = s.forecastLine(ctx, placed[0], day, us, parts)
	}
	if s.Cfg.WeatherAlerts() {
		seen := map[string]bool{}
		for _, p := range placed {
			found, err := s.alertsAt(ctx, p.Lat, p.Lon)
			for _, a := range found {
				if err == nil && nws.ForDriving(a) && !seen[a.Event] && (a.Ends == 0 || a.Ends > day.Unix()) && (a.Onset == 0 || a.Onset < day.AddDate(0, 0, 1).Unix()) {
					seen[a.Event] = true
					parts = append(parts, fmt.Sprintf("%s at %s.", a.Event, p.Name))
				}
			}
		}
	}
	if line := s.batteryLine(placed, lowC, us); line != "" {
		parts = append(parts, line)
	}
	return title, strings.Join(parts, " "), true
}

// forecastLine adds the day's low and any snow or rain at the first appointment (7 am to 7 pm).
func (s *Server) forecastLine(ctx context.Context, p *companyPick, day time.Time, us bool, parts []string) (float64, []string) {
	c := s.Weather
	if c == nil {
		c = &weather.Client{}
	}
	hours, err := c.Day(ctx, p.Lat, p.Lon, day)
	if err != nil {
		return math.NaN(), parts
	}
	low, snow, rain := math.Inf(1), 0.0, 0.0
	for _, h := range hours {
		hr := time.Unix(h.TS, 0).In(s.Cfg.Location()).Hour()
		if hr < 7 || hr >= 19 {
			continue
		}
		if h.TempC != nil && *h.TempC < low {
			low = *h.TempC
		}
		snow += h.SnowCm
		rain += h.RainMm
	}
	if math.IsInf(low, 1) {
		return math.NaN(), parts
	}
	line := fmt.Sprintf("Low %.0f°C", low)
	if us {
		line = fmt.Sprintf("Low %.0f°F", low*9/5+32)
	}
	switch {
	case snow >= 0.5 && us:
		line += fmt.Sprintf(", about %.1f in of snow", snow/2.54)
	case snow >= 0.5:
		line += fmt.Sprintf(", about %.0f cm of snow", snow)
	case rain >= 1:
		line += ", rain"
	}
	return low, append(parts, line+".")
}

// batteryLine is how much of the battery the day takes (Home, the appointments in order, Home) at the day's temperature, and what to
// charge to with the reserve kept. Empty for a car with no battery level, or with no way to tell its range.
func (s *Server) batteryLine(placed []*companyPick, lowC float64, us bool) string {
	x, _ := s.S.LatestSample()
	home := s.homeNear()
	if x == nil || x.SoCPct == nil || home == nil || len(placed) == 0 {
		return ""
	}
	km, at := 0.0, charging.Point{Lat: home[0], Lon: home[1]}
	for _, p := range append(placed, &companyPick{Lat: home[0], Lon: home[1]}) {
		next := charging.Point{Lat: p.Lat, Lon: p.Lon}
		km += charging.DistanceM(at, next) / 1000 * roadFactor
		at = next
	}
	per := s.kmPerPctAt(lowC)
	if per <= 0 || km < 1 {
		return ""
	}
	need := km / per
	reserve := 20.0
	target := math.Min(100, math.Ceil((need+reserve)/5)*5)
	dist := fmt.Sprintf("%.0f km", km)
	if us {
		dist = fmt.Sprintf("%.0f mi", km/1.609344)
	}
	line := fmt.Sprintf("The day is about %s, about %.0f%% of the battery.", dist, need)
	if *x.SoCPct >= target {
		return line + fmt.Sprintf(" The battery at %.0f%% is enough.", *x.SoCPct)
	}
	return line + fmt.Sprintf(" Charge to at least %.0f%% to keep %.0f%% spare.", target, reserve)
}

// kmPerPctAt is how far the car goes per percent of battery at a temperature, from its own trips in that weather, else its recent
// trips, else its own range figure. 0 when none is known.
func (s *Server) kmPerPctAt(tempC float64) float64 {
	now := time.Now()
	year, _ := s.S.BandTrips(now.AddDate(0, 0, -370).Unix())
	if !math.IsNaN(tempC) {
		if temps, err := s.S.TripTemps(now.AddDate(0, 0, -370).Unix()); err == nil {
			if b := trips.BandFor(tempC); b != nil {
				for _, w := range trips.MeasureByBand(year, temps) {
					if per, ok := w.KmPerPct(); ok && w.Band.Name == b.Name {
						return per
					}
				}
			}
		}
	}
	var recent = year[:0:0]
	for _, t := range year {
		if t.StartTS >= now.AddDate(0, 0, -30).Unix() {
			recent = append(recent, t)
		}
	}
	if per, ok := trips.Measure(recent).KmPerPct(); ok {
		return per
	}
	if x, _ := s.S.LatestSample(); x != nil && x.RangeKm != nil && x.SoCPct != nil && *x.SoCPct >= 10 {
		return *x.RangeKm / *x.SoCPct
	}
	return 0
}
