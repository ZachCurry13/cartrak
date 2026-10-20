package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/debugreport"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// withPackPower fills in the power of fast-charging readings, which the car reports as zero, from the pack's current and
// voltage.
func (s *Server) withPackPower(rows []store.DebugRow, from, to int64) {
	cs, err := s.S.ChargingSamples(from, to)
	if err != nil || len(cs) == 0 {
		return
	}
	byTS := map[int64]float64{}
	for _, x := range cs {
		if kw := samplePower(x); kw > 0.2 {
			byTS[x.TS] = kw
		}
	}
	for i := range rows {
		if kw, ok := byTS[rows[i].TS]; ok {
			v := kw
			rows[i].PackKw = &v
		}
	}
}

// chargeCurves is, for each session, the power as the battery passed 20, 40, 60 and 80 percent.
func (s *Server) chargeCurves(charges []*store.Charge) map[int64]string {
	out := map[int64]string{}
	for _, c := range charges {
		if c.EndTS == nil {
			continue
		}
		rows, err := s.S.SamplesBetween(c.StartTS, *c.EndTS)
		if err != nil || len(rows) < 3 {
			continue
		}
		var parts []string
		for _, lvl := range []float64{20, 40, 60, 80} {
			var prev *float64
			for _, x := range rows {
				if x.SoCPct == nil {
					continue
				}
				if *x.SoCPct >= lvl {
					// only levels the session climbed through: it must have started below it
					if prev != nil && *prev < lvl {
						if kw := samplePower(x); kw > 0.2 {
							parts = append(parts, fmt.Sprintf("%.0f%% %.0f kW", lvl, kw))
						}
					}
					break
				}
				prev = x.SoCPct
			}
		}
		if len(parts) > 0 {
			out[c.ID] = strings.Join(parts, ", ")
		}
	}
	return out
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// debugSettings lists what shapes behaviour. Secrets and positions are never included.
func (s *Server) debugSettings() []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	if s.P != nil {
		add("polling: driving every %d s, charging every %d s, parked every %d s", int(s.P.Iv.Driving.Seconds()), int(s.P.Iv.Charging.Seconds()), int(s.P.Iv.Parked.Seconds()))
	}
	if s.Cfg != nil {
		price := func(p *float64) string {
			if p == nil {
				return "not set"
			}
			return fmt.Sprintf("%s%.3f", s.Cfg.Currency(), *p)
		}
		add("prices per kWh: home %s, public %s; slow-charge efficiency %.0f%%", price(s.Cfg.PriceHome()), price(s.Cfg.PricePublic()), s.Cfg.ChargeEfficiency()*100)
		add("alerts: low battery %.0f%% (0 is off), bump level %.1f (0 is off), plug-in reminder %q below %.0f%%", s.Cfg.LowBatteryLevel(), s.Cfg.BumpThreshold(), s.Cfg.PlugReminderTime(), s.Cfg.PlugReminderBelow())
		add("lookups: wind %s, road estimate %s, maps %s, city names %s", onOff(s.Cfg.WeatherLookup()), onOff(s.Cfg.RouteLookup()), onOff(s.Cfg.MapEnabled()), onOff(s.Cfg.GeocodeEnabled()))
		add("time zone: %s", s.Cfg.Location())
	}
	vi := s.vehicleInfo()
	add("vehicle: %s (%s), battery %s", vi.Label, vi.Powertrain, capText(s.capacityKwh()))
	if named, err := s.S.ListNamedPlaces(); err == nil {
		var ps []string
		for _, p := range named {
			ps = append(ps, fmt.Sprintf("%s (%.0f m)", p.Name, p.RadiusM))
		}
		add("named places (%d): %s", len(named), strings.Join(ps, ", "))
	}
	add("%s", s.debugInstallation())
	if cats, err := s.S.ListCategories(); err == nil {
		custom := 0
		for _, c := range cats {
			if !c.Builtin {
				custom++
			}
		}
		mode := s.S.TimeOffKind(time.Now().In(s.defaultLoc()).Format("2006-01-02"))
		if mode == "" {
			mode = "none"
		}
		add("categories: %d (%d of your own); mode today: %s", len(cats), custom, mode)
	}
	home := "none"
	if s.namedHome() != nil {
		home = "a place named Home"
	} else if s.homePoint() != nil {
		home = "a session marked as Home"
	}
	priced := 0
	if named, err := s.S.ListNamedPlaces(); err == nil {
		for _, p := range named {
			if p.ChargePrice != nil || p.ChargeHome {
				priced++
			}
		}
	}
	add("home is known from: %s; places with their own charging price or counted as home: %d", home, priced)
	add("trips with a corrected distance: %d; trips whose road could not be estimated: %d", s.S.CountCorrectedTrips(), s.S.CountNoRoutes())
	if shares, err := s.S.ListShares(); err == nil {
		live := 0
		now := time.Now().Unix()
		for _, x := range shares {
			if !x.Revoked && (x.Expires == 0 || x.Expires > now) {
				live++
			}
		}
		add("public share links in use: %d", live)
	}
	if ts, err := s.S.ListReportTemplates(); err == nil {
		add("custom reports: %d", len(ts))
	}
	add("data window: first reading %s", time.Unix(s.S.FirstSampleTS(), 0).In(s.defaultLoc()).Format("2006-01-02"))
	return out
}

func capText(c *float64) string {
	if c == nil {
		return "size unknown"
	}
	return fmt.Sprintf("%.1f kWh", *c)
}

func toPollInfo(ps []poller.PollRec) []debugreport.PollInfo {
	out := make([]debugreport.PollInfo, 0, len(ps))
	for _, p := range ps {
		out = append(out, debugreport.PollInfo{TS: p.TS, Result: p.Result, NextSec: p.NextSec, Detail: p.Detail})
	}
	return out
}

// debugInstallation says where this car sits among the others, without naming anyone or any car.
func (s *Server) debugInstallation() string {
	cars, _ := s.core().ListCars()
	if s.CarID == 0 || len(cars) == 0 {
		return "installation: a single car"
	}
	pos, owner := 0, "the first car"
	for i, c := range cars {
		if c.ID == s.CarID {
			pos = i + 1
			if c.OwnerID != 0 {
				owner = "added by a person"
			}
		}
	}
	return fmt.Sprintf("installation: car %d of %d (%s)", pos, len(cars), owner)
}
