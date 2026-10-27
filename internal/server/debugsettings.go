package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// debugSettings lists what shapes behavior. Secrets and positions are never included.
func (s *Server) debugSettings() []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	if l, ok := s.Link.(vehicleLister); ok {
		var all []string
		for _, v := range l.Vehicles() {
			all = append(all, v.Label+" ending "+vinEnd(v.VIN))
		}
		add("Ford vehicle followed: ending %s (chosen: %v); the linked account lists: %s", vinEnd(s.S.GetKV(ford.SeenVINKey)), s.Cfg != nil && s.Cfg.Value("FORD_VIN") != "", strings.Join(all, ", "))
	}
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
		add("alerts: battery reserve %.0f%%, bump level %.1f (0 is off), plug-in reminder %q below %.0f%%", s.Cfg.LowBatteryLevel(), s.Cfg.BumpThreshold(), s.Cfg.PlugReminderTime(), s.Cfg.PlugReminderBelow())
		add("lookups: wind %s, road estimate %s, maps %s, city names %s", onOff(s.Cfg.WeatherLookup()), onOff(s.Cfg.RouteLookup()), onOff(s.Cfg.MapEnabled()), onOff(s.Cfg.GeocodeEnabled()))
		key := "no key"
		if s.Cfg.EIAKey() != "" {
			key = "key set"
		}
		add("gas comparison: %s; live gas prices %s (%s, %d place adjustments), US units %v", onOff(s.Cfg.Gas().Set()), onOff(s.liveGasOn()), key, len(s.Cfg.GasAdjusts()), s.Cfg.LiveGasUnits())
		if s.liveGasOn() {
			st := s.gasService().Status()
			last, errText := "never", "none"
			if st.OK > 0 {
				last = ago(time.Now().Unix() - st.OK)
			}
			if st.Err != "" {
				errText = st.Err
			}
			add("live gas prices: last updated %s, last error: %s", last, errText)
		}
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
