package trips

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

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
		d.backfillStart(c, st)
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
	if err := d.S.UpdateCharge(open); err != nil {
		return err
	}
	if !open.Open {
		d.noteChargerKwh(open, st)
	}
	return nil
}

// Reclassify re-runs the rules over every trip that was not set by hand.
func (d *Detector) Reclassify() error {
	ts, err := d.S.ListTrips(store.TripFilter{Limit: 1000})
	if err != nil {
		return err
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].StartTS < ts[j].StartTS }) // oldest first: an outing depends on the trips before it
	for i, t := range ts {
		if t.CategorySource == "manual" {
			continue
		}
		from := i
		for from > 0 && t.StartTS-ts[from-1].StartTS < priorWindowSec {
			from--
		}
		cat, rid, src := d.classifyAfter(t.StartTS, t.StartLat, t.StartLon, d.tripZone(t), ts[from:i])
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
	d.ClassifyTripAfter(t, nil)
}

// ClassifyTripAfter is ClassifyTrip with the trips before it given, oldest first (nil reads them from the store).
func (d *Detector) ClassifyTripAfter(t *store.Trip, prior []*store.Trip) {
	t.Category, t.RuleID, t.CategorySource = d.classifyAfter(t.StartTS, t.StartLat, t.StartLon, d.tripZone(t), prior)
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
