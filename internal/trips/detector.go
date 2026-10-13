package trips

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tzlookup"
)

// Detector opens and closes trips and charge sessions from successive readings.
type Detector struct {
	S    *store.Store
	TZ   *time.Location
	TZFn func() *time.Location // optional: read the zone live (admin panel); wins over TZ
}

// Driving reports whether a reading looks like the car is in use.
func Driving(st *ford.State) bool { return st.Driving() }

func charging(st *ford.State) bool { return st.Charging() }

// Process applies one reading. Call it once per poll, oldest first.
func (d *Detector) Process(st *ford.State) error {
	if err := d.trip(st); err != nil {
		return err
	}
	return d.charge(st)
}

// zone is the time zone where a trip started: found from its location, else the
// default time zone setting. So "weekdays 8am-5pm" means 8 to 5 wherever you are driving.
func (d *Detector) zone(lat, lon *float64) *time.Location {
	if lat != nil && lon != nil {
		return tzlookup.Location(*lat, *lon, d.loc())
	}
	return d.loc()
}

// tripZone prefers the zone stored with the trip over looking it up again.
func (d *Detector) tripZone(t *store.Trip) *time.Location {
	if t.TZ != "" {
		if l, err := time.LoadLocation(t.TZ); err == nil {
			return l
		}
	}
	return d.zone(t.StartLat, t.StartLon)
}

func (d *Detector) classify(startTS int64, lat, lon *float64, loc *time.Location) (string, *int64, string) {
	rules, err := d.S.ListRules()
	if err == nil {
		at := time.Unix(startTS, 0).In(loc)
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
		t := &store.Trip{StartTS: now, StartLat: st.Lat, StartLon: st.Lon, StartOdo: st.OdometerKm, StartSoC: st.SoCPct, StartKwh: st.EnergyKwh}
		var rid *int64
		loc := d.zone(st.Lat, st.Lon)
		t.TZ = loc.String()
		t.Category, rid, t.CategorySource = d.classify(now, st.Lat, st.Lon, loc)
		t.RuleID = rid
		return d.S.CreateTrip(t)
	}
	if open == nil {
		return nil
	}
	open.EndLat, open.EndLon, open.EndOdo, open.EndSoC = pick(st.Lat, open.EndLat), pick(st.Lon, open.EndLon),
		pick(st.OdometerKm, open.EndOdo), pick(st.SoCPct, open.EndSoC)
	open.EndKwh = pick(st.EnergyKwh, open.EndKwh)
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
	ss, err := d.S.TrackBetween(t.StartTS, *t.EndTS)
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
		c := &store.Charge{StartTS: now, Lat: st.Lat, Lon: st.Lon, StartSoC: st.SoCPct, StartKwh: st.EnergyKwh}
		if st.ChargeKw != nil {
			c.PeakKw = *st.ChargeKw
		}
		return d.S.CreateCharge(c)
	}
	if open == nil {
		return nil
	}
	open.EndTS, open.EndSoC, open.EndKwh = &now, pick(st.SoCPct, open.EndSoC), pick(st.EnergyKwh, open.EndKwh)
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
		cat, rid, src := d.classify(t.StartTS, t.StartLat, t.StartLon, d.tripZone(t))
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
