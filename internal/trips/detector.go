package trips

import (
	"fmt"
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

	lastOdo  *float64 // the odometer at the previous reading, to tell a car that has moved from one that only woke up
	sawSpeed bool     // this feed has sent a speed before, so a reading without one is a gap, not a feed that has none

	lastReading    int64    // time of the previous reading this detector saw
	lastDisconnect int64    // the last time a reading showed the cable unplugged
	lastKw         float64  // the charging power the previous reading reported
	lastSoC        *float64 // the previous reading's battery level and energy, and whether it was charging
	lastKwh        *float64
	lastCharging   bool
	autoKwh        map[int64]float64 // the charger kWh this detector filled in itself, by session
}

// Driving reports whether a reading looks like the car is in use. A car that is taking charge and not moving is
// parked, even if the ignition was left on: otherwise a road trip with the car left on at a fast charger becomes one
// long "trip" whose energy use nets out the charge it took.
func Driving(st *ford.State) bool {
	if st.Charging() && !(st.SpeedKph != nil && *st.SpeedKph > 3) {
		return false
	}
	return st.Driving()
}

func charging(st *ford.State) bool { return st.Charging() }

// Process applies one reading. Call it once per poll, oldest first.
func (d *Detector) Process(st *ford.State) error {
	defer func() {
		d.lastReading = st.TS.Unix()
		d.lastKw = 0
		d.lastSoC, d.lastKwh, d.lastCharging = st.SoCPct, st.EnergyKwh, charging(st)
		if st.ChargeKw != nil {
			d.lastKw = *st.ChargeKw
		}
	}()
	if st.Plug != "" && !st.PlugConnected() {
		d.lastDisconnect = st.TS.Unix()
	}
	err := d.trip(st)
	if st.OdometerKm != nil {
		d.lastOdo = st.OdometerKm
	}
	if st.SpeedKph != nil {
		d.sawSpeed = true
	}
	if err != nil {
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
	// A vacation or a work trip wins over the rules: no manual relabelling of every trip for a week away. A trip the owner
	// set by hand is never classified at all, so it still wins over this.
	switch kind := d.S.TimeOffKind(time.Unix(startTS, 0).In(loc).Format("2006-01-02")); {
	case kind == store.KindVacation:
		return Personal, nil, "vacation"
	case kind != "" && d.S.CategoryExists(kind):
		return kind, nil, "mode"
	}
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
	if Driving(st) && open == nil && moving(st) && d.leftTheSpot(st) {
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
		if (open.DistanceKm < 0.3 && now-open.StartTS < 120) || (open.DistanceKm < 0.2 && open.MaxSpeedKph < 8) {
			d.S.LogDecision("trip_discarded", fmt.Sprintf("trip %d thrown away as a blip: %.2f mi (%.2f km) in %d s, top speed %.0f mph (%.0f km/h)", open.ID, open.DistanceKm/1.609344, open.DistanceKm, now-open.StartTS, open.MaxSpeedKph/1.609344, open.MaxSpeedKph))
			return d.S.DeleteTrip(open.ID) // an ignition blip, or sitting in the car with it on: not a trip
		}
	}
	return d.S.UpdateTrip(open)
}

// moving says a trip should begin: the car is actually going somewhere. Sitting in the car with the ignition or
// accessories on (speed zero) is not a trip. When a feed gives no speed at all, ignition on has to be enough.
func moving(st *ford.State) bool {
	if st.SpeedKph == nil {
		// A remote start turns the ignition on with nobody driving; the car says so with a countdown until it shuts off.
		if t := ford.Number(st.Extra, "remoteStartCountdownTimer"); t != nil && *t > 0 {
			return false
		}
		return true
	}
	return *st.SpeedKph > 3
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
