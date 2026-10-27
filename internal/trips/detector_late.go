package trips

import (
	"fmt"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// lateKmh is the average speed assumed to place a drive nobody saw inside the silence it happened in.
const lateKmh = 40.0

// A parked car is asked about every ten minutes, so a short hop between two customers can start and end between two readings,
// with the car never reported moving. The odometer still shows it: the first reading afterwards is further on than the highest
// one before it. That is a drive, and it is recorded as one, ending at the reading that revealed it.

// missedRise is how far the odometer has gone since the highest reading before this one, when that is a drive worth recording: at
// least a kilometer, at a speed a car can do in the time that passed. Otherwise 0.
func (d *Detector) missedRise(st *ford.State) float64 {
	if st.OdometerKm == nil || d.maxOdo == nil || d.lastReading == 0 {
		return 0
	}
	rise, gap := *st.OdometerKm-*d.maxOdo, float64(st.TS.Unix()-d.lastReading)
	if rise < 1 || gap <= 0 || rise/(gap/3600) > 200 {
		return 0
	}
	return rise
}

// lateTrip starts the trip for a drive that was only seen at its end. It begins as late as the distance allows (at lateKmh), but
// never before the reading it follows; the caller finishes and closes it with the reading in hand.
func (d *Detector) lateTrip(st *ford.State, rise float64) (*store.Trip, error) {
	now := st.TS.Unix()
	start := now - int64(rise/lateKmh*3600)
	if start < d.lastReading {
		start = d.lastReading
	}
	t := &store.Trip{StartTS: start, StartOdo: d.maxOdo, StartSoC: d.lastSoC, StartKwh: d.lastKwh}
	t.StartLat, t.StartLon = d.S.LastPositionBefore(start)
	loc := d.zone(t.StartLat, t.StartLon)
	t.TZ = loc.String()
	var rid *int64
	t.Category, rid, t.CategorySource = d.classify(start, t.StartLat, t.StartLon, loc)
	t.RuleID = rid
	d.S.LogDecision("trip_late", fmt.Sprintf("the car was first seen %.1f km further on than the highest reading before it, %d min earlier, and no reading showed it moving: a drive between the two was recorded", rise, (now-d.lastReading)/60))
	return t, d.S.CreateTrip(t)
}
