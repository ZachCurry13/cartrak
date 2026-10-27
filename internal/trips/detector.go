package trips

import (
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Detector opens and closes trips and charge sessions from successive readings.
type Detector struct {
	S    *store.Store
	TZ   *time.Location
	TZFn func() *time.Location // optional: read the zone live (admin panel); wins over TZ
	Work func() WorkDay        // optional: the "first outing of a work day is work" rule, read live; nil leaves it off

	maxOdo   *float64 // the highest odometer seen: an old lower value on one reading is not a car that went back
	lastOdo  *float64 // the odometer at the previous reading, to tell a car that has moved from one that only woke up
	sawSpeed bool     // this feed has sent a speed before, so a reading without one is a gap, not a feed that has none

	endlessTrip    int64 // a trip just closed with no end position, waiting for the car to report one, and when it closed
	endlessAt      int64
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
	if st.Blank() { // an empty answer (or an old stored one) says nothing about the car: it must not end a drive
		return nil
	}
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
	err := d.settleEnd(st)
	if err == nil {
		err = d.trip(st)
	}
	if st.OdometerKm != nil {
		d.lastOdo = st.OdometerKm
		if d.maxOdo == nil || *st.OdometerKm > *d.maxOdo {
			d.maxOdo = st.OdometerKm
		}
	}
	if st.SpeedKph != nil {
		d.sawSpeed = true
	}
	if err != nil {
		return err
	}
	return d.charge(st)
}

func (d *Detector) trip(st *ford.State) error {
	open, err := d.S.OpenTrip()
	if err != nil {
		return err
	}
	now := st.TS.Unix()
	if Driving(st) && open == nil && moving(st) && d.leftTheSpot(st) {
		startLat, startLon := st.Lat, st.Lon
		if startLat == nil || startLon == nil { // the car did not say where it began: it began where it was last seen parked
			startLat, startLon = d.S.LastPositionBefore(now)
		}
		t := &store.Trip{StartTS: now, StartLat: startLat, StartLon: startLon, StartOdo: st.OdometerKm, StartSoC: st.SoCPct, StartKwh: st.EnergyKwh}
		var rid *int64
		loc := d.zone(startLat, startLon)
		t.TZ = loc.String()
		t.Category, rid, t.CategorySource = d.classify(now, startLat, startLon, loc)
		t.RuleID = rid
		return d.S.CreateTrip(t)
	}
	missed := false
	if open == nil {
		if rise := d.missedRise(st); rise > 0 {
			t, err := d.lateTrip(st, rise)
			if err != nil {
				return err
			}
			open, missed = t, true
		}
	}
	if open == nil {
		return nil
	}
	open.EndLat, open.EndLon, open.EndOdo, open.EndSoC = pick(st.Lat, open.EndLat), pick(st.Lon, open.EndLon),
		pickOdo(st.OdometerKm, open.EndOdo), pick(st.SoCPct, open.EndSoC)
	open.EndKwh = pick(st.EnergyKwh, open.EndKwh)
	if st.SpeedKph != nil && *st.SpeedKph > open.MaxSpeedKph {
		open.MaxSpeedKph = *st.SpeedKph
	}
	open.EndTS = &now
	open.DistanceKm = d.distance(open)
	if missed {
		open.Open = false // seen only at its end: it is over
	}
	if !Driving(st) {
		open.Open = false
		if (open.DistanceKm < 0.3 && now-open.StartTS < 120) || (open.DistanceKm < 0.2 && open.MaxSpeedKph < 8) {
			d.S.LogDecision("trip_discarded", fmt.Sprintf("trip %d thrown away as a blip: %.2f mi (%.2f km) in %d s, top speed %.0f mph (%.0f km/h)", open.ID, open.DistanceKm/1.609344, open.DistanceKm, now-open.StartTS, open.MaxSpeedKph/1.609344, open.MaxSpeedKph))
			return d.S.DeleteTrip(open.ID) // an ignition blip, or sitting in the car with it on: not a trip
		}
	}
	if !open.Open && staleEnd(open) {
		open.EndLat, open.EndLon = nil, nil
		d.endlessTrip, d.endlessAt = open.ID, now
	}
	if err := d.S.UpdateTrip(open); err != nil {
		return err
	}
	d.endOuting(open)
	return nil
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
