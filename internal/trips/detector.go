package trips

import (
	"fmt"
	"strings"
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

	lastReading    int64   // time of the previous reading this detector saw
	lastDisconnect int64   // the last time a reading showed the cable unplugged
	lastKw         float64 // the charging power the previous reading reported
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
	// Vacation wins over the rules: no manual relabelling of every trip for a week away. A trip the owner set by hand
	// is never classified at all, so it still wins over this.
	if d.S.OnTimeOff(time.Unix(startTS, 0).In(loc).Format("2006-01-02")) {
		return Personal, nil, "vacation"
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

func (d *Detector) charge(st *ford.State) error {
	open, err := d.S.OpenCharge()
	if err != nil {
		return err
	}
	now := st.TS.Unix()
	if charging(st) && open == nil {
		if prev := d.resumable(st); prev != nil {
			// A pause (the charger handing over, a hiccup) is not the end of a session. Pick it up again, so the energy that went
			// in during the gap is counted and the session is not split in two.
			if err := d.S.ReopenCharge(prev.ID); err != nil {
				return err
			}
			prev.Open = true
			d.S.LogDecision("charge_resumed", fmt.Sprintf("charging session %d picked up again after a pause of %d min (cable still connected)", prev.ID, (now-*prev.EndTS)/60))
			open = prev
		}
	}
	if charging(st) && open == nil {
		c := &store.Charge{StartTS: now, Lat: st.Lat, Lon: st.Lon, StartSoC: st.SoCPct, StartKwh: st.EnergyKwh}
		if st.ChargeKw != nil {
			c.PeakKw = *st.ChargeKw
		}
		c.Station = stationOf(st)
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
	if dt := now - d.lastReading; d.lastReading > 0 && st.ChargeKw != nil && dt > 0 && dt <= maxPowerGap {
		kwh := (d.lastKw + *st.ChargeKw) / 2 * float64(dt) / 3600 // the power between two readings, as a straight line
		sum := kwh
		if open.PowerKwh != nil {
			sum += *open.PowerKwh
		}
		open.PowerKwh = &sum
	}
	if s := stationOf(st); s != "" {
		open.Station = s
	}
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

// ClassifyTrip sets a trip's category from the rules, as it would be when the trip is first recorded.
func (d *Detector) ClassifyTrip(t *store.Trip) {
	t.Category, t.RuleID, t.CategorySource = d.classify(t.StartTS, t.StartLat, t.StartLon, d.tripZone(t))
}

// stationOf is the kind of charger the car reports, but only while it is actually charging: the field keeps its last
// value after the cable is pulled, so it would otherwise name the charger from the last session.
func stationOf(st *ford.State) string {
	if !charging(st) {
		return ""
	}
	v := strings.ToUpper(strings.TrimSpace(st.StationType))
	if v == "" || v == "NOT_DETECTED" || v == "UNKNOWN" || v == "INVALID" {
		return ""
	}
	return v
}

// leftTheSpot is for a reading that carries no speed although the feed normally does, such as the first one after the car
// wakes: a trip then starts only when the odometer shows the car has moved since the reading before. A feed that never
// sends a speed, or an odometer, still has to take the ignition at its word.
func (d *Detector) leftTheSpot(st *ford.State) bool {
	if st.SpeedKph != nil || !d.sawSpeed || st.OdometerKm == nil {
		return true
	}
	return d.lastOdo != nil && *st.OdometerKm-*d.lastOdo >= 0.1
}

// resumeWithin is how long a charging session can stay paused and still be the same session when charging resumes.
const resumeWithin = 20 * 60

// maxPowerGap is the longest gap between two readings that is still counted as charging when adding up the power; a
// longer silence says nothing about what happened in it.
const maxPowerGap = 15 * 60

// replugWithin is how long the cable can be out and back in, at the same place, and still be the same session. A fast
// charger that stops or faults often gets exactly that: the driver pulls the plug and puts it back.
const replugWithin = 5 * 60

// resumable is the session that has just paused and that charging has now resumed in: it ended no more than twenty minutes
// ago, this detector watched the whole gap, the cable was not unplugged in between (or only for a quick replug), and the
// car has not moved.
func (d *Detector) resumable(st *ford.State) *store.Charge {
	if !st.PlugConnected() {
		return nil
	}
	list, err := d.S.ListCharges(1) // newest first
	if err != nil || len(list) == 0 {
		return nil
	}
	prev := list[0]
	now := st.TS.Unix()
	if prev.Open || prev.EndTS == nil || now-*prev.EndTS > resumeWithin || now < *prev.EndTS {
		return nil
	}
	if d.lastReading < *prev.EndTS {
		return nil // not watched
	}
	if d.lastDisconnect >= *prev.EndTS { // unplugged in between: only a quick replug with the battery not lower counts
		if now-*prev.EndTS > replugWithin || (prev.EndSoC != nil && st.SoCPct != nil && *st.SoCPct < *prev.EndSoC-2) {
			return nil
		}
	}
	if prev.Lat != nil && prev.Lon != nil && st.Lat != nil && st.Lon != nil && Haversine(*prev.Lat, *prev.Lon, *st.Lat, *st.Lon) > 150 {
		return nil // somewhere else
	}
	return prev
}
