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

// chargeCostFrom says, for each recent charge, where its cost came from and what a kWh cost, for the debug report.
func (s *Server) chargeCostFrom(from time.Time) map[int64]string {
	out := map[int64]string{}
	sessions, err := s.chargeSessions(300)
	if err != nil {
		return out
	}
	for _, c := range sessions {
		if c.EndTS != nil && *c.EndTS < from.Unix() {
			continue
		}
		switch {
		case c.Cost == nil:
			out[c.ID] = "not priced (no price set for " + dashIf(c.Where) + ")"
		case c.WallKwh != nil && *c.WallKwh > 0.5:
			out[c.ID] = fmt.Sprintf("%s, %.2f a kWh delivered (%.1f kWh at the charger), %s", c.PriceFrom, *c.Cost / *c.WallKwh, *c.WallKwh, dashIf(c.Where))
		default:
			out[c.ID] = fmt.Sprintf("%s, %.2f in all, %s", c.PriceFrom, *c.Cost, dashIf(c.Where))
		}
	}
	return out
}

func dashIf(s string) string {
	if s == "" {
		return "place unknown"
	}
	return s
}

// awayDays lists the time-away stretches that touch the report and, for each day, how its base was settled: chosen, found from
// where the car slept, or none (at home, or not yet known). No positions or place names.
func (s *Server) awayDays(from, now time.Time) []string {
	loc := s.defaultLoc()
	bases := s.baseFinder(nil)
	var out []string
	seen := map[int64]bool{}
	for d := from.In(loc); !d.After(now); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		st := s.S.StretchOn(day)
		if st == nil {
			continue
		}
		if !seen[st.ID] {
			seen[st.ID] = true
			out = append(out, fmt.Sprintf("stretch %d: %s to %s, %s, base %s", st.ID, st.From, st.To, st.Kind, map[bool]string{true: "chosen by you", false: "found from the nights"}[st.BaseID != nil]))
		}
		b := bases.forDay(day)
		how := "no base (the car slept at home, or no night position yet)"
		if b != nil {
			how = "base found"
			if b.Name != "" {
				how = "base found, named"
			}
		}
		out = append(out, "  "+day+": "+how)
	}
	return out
}

// vinEnd is the last six characters of a VIN: enough to tell vehicles apart in a report without printing the VIN.
func vinEnd(vin string) string {
	if len(vin) > 6 {
		return vin[len(vin)-6:]
	}
	if vin == "" {
		return "?"
	}
	return vin
}
