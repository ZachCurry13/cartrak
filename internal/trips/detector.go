package trips

import (
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Detector opens and closes trips and charge sessions from successive readings.
type Detector struct {
	S    *store.Store
	TZ   *time.Location
	TZFn func() *time.Location // optional: read the zone live (admin panel); wins over TZ
}

// Driving reports whether a reading looks like the car is in use.
func Driving(st *ford.State) bool {
	switch strings.ToUpper(st.Ignition) {
	case "ON", "RUN", "START", "STARTING", "RUNNING":
		return true
	}
	return st.SpeedKph != nil && *st.SpeedKph > 3
}

func charging(st *ford.State) bool {
	return strings.Contains(strings.ToLower(st.ChargeState), "charging") ||
		(st.ChargeKw != nil && *st.ChargeKw > 0.2)
}

// Process applies one reading. Call it once per poll, oldest first.
func (d *Detector) Process(st *ford.State) error {
	if err := d.trip(st); err != nil {
		return err
	}
	return d.charge(st)
}

func (d *Detector) classify(startTS int64, lat, lon *float64) (string, *int64, string) {
	rules, err := d.S.ListRules()
	if err == nil {
		at := time.Unix(startTS, 0).In(d.loc())
		if cat, id, ok := Classify(rules, at, lat, lon); ok {
			return cat, &id, "rule"
		}
	}
	return Personal, nil, "default"
}

func (d *Detector) trip(st *ford.State) error {
	open, err := d.S.OpenTrip()
	if err != nil {
		return err
	}
	now := st.TS.Unix()
	if Driving(st) && open == nil {
		t := &store.Trip{StartTS: now, StartLat: st.Lat, StartLon: st.Lon, StartOdo: st.OdometerKm, StartSoC: st.SoCPct}
		var rid *int64
		t.Category, rid, t.CategorySource = d.classify(now, st.Lat, st.Lon)
		t.RuleID = rid
		return d.S.CreateTrip(t)
	}
	if open == nil {
		return nil
	}
	open.EndLat, open.EndLon, open.EndOdo, open.EndSoC = pick(st.Lat, open.EndLat), pick(st.Lon, open.EndLon),
		pick(st.OdometerKm, open.EndOdo), pick(st.SoCPct, open.EndSoC)
	if st.SpeedKph != nil && *st.SpeedKph > open.MaxSpeedKph {
		open.MaxSpeedKph = *st.SpeedKph
	}
	open.EndTS = &now
	open.DistanceKm = d.distance(open)
	if !Driving(st) {
		open.Open = false
		if open.DistanceKm < 0.3 && now-open.StartTS < 120 {
			return d.S.DeleteTrip(open.ID) // ignition blip, not a trip
		}
	}
	return d.S.UpdateTrip(open)
}

func pick(n, old *float64) *float64 {
	if n != nil {
		return n
	}
	return old
}

// distance prefers the odometer; otherwise it sums GPS hops between samples.
func (d *Detector) distance(t *store.Trip) float64 {
	if t.StartOdo != nil && t.EndOdo != nil && *t.EndOdo >= *t.StartOdo {
		return *t.EndOdo - *t.StartOdo
	}
	if t.EndTS == nil {
		return 0
	}
	ss, err := d.S.SamplesBetween(t.StartTS, *t.EndTS)
	if err != nil {
		return 0
	}
	var m float64
	var pl, po *float64
	for _, s := range ss {
		if s.Lat == nil || s.Lon == nil {
			continue
		}
		if pl != nil {
			m += Haversine(*pl, *po, *s.Lat, *s.Lon)
		}
		pl, po = s.Lat, s.Lon
	}
	return m / 1000
}

func (d *Detector) charge(st *ford.State) error {
	open, err := d.S.OpenCharge()
	if err != nil {
		return err
	}
	now := st.TS.Unix()
	if charging(st) && open == nil {
		c := &store.Charge{StartTS: now, Lat: st.Lat, Lon: st.Lon, StartSoC: st.SoCPct}
		if st.ChargeKw != nil {
			c.PeakKw = *st.ChargeKw
		}
		return d.S.CreateCharge(c)
	}
	if open == nil {
		return nil
	}
	open.EndTS, open.EndSoC = &now, pick(st.SoCPct, open.EndSoC)
	if st.ChargeKw != nil && *st.ChargeKw > open.PeakKw {
		open.PeakKw = *st.ChargeKw
	}
	open.Open = charging(st)
	return d.S.UpdateCharge(open)
}

// Reclassify re-runs the rules over every trip that was not set by hand.
func (d *Detector) Reclassify() error {
	ts, err := d.S.ListTrips(store.TripFilter{Limit: 1000})
	if err != nil {
		return err
	}
	for _, t := range ts {
		if t.CategorySource == "manual" {
			continue
		}
		cat, rid, src := d.classify(t.StartTS, t.StartLat, t.StartLon)
		t.Category, t.RuleID, t.CategorySource = cat, rid, src
		if err := d.S.UpdateTrip(t); err != nil {
			return err
		}
	}
	return nil
}

func (d *Detector) loc() *time.Location {
	if d.TZFn != nil {
		return d.TZFn()
	}
	if d.TZ != nil {
		return d.TZ
	}
	return time.UTC
}
