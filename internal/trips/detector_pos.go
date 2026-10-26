package trips

import (
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// pickOdo keeps the highest odometer reading of a trip. A car can report an old, lower value on its last reading (the ignition-off one),
// which would turn a drive into 0 miles.
func pickOdo(n, old *float64) *float64 {
	if n != nil && (old == nil || *n >= *old) {
		return n
	}
	return old
}

// staleEnd says a finished trip's end position is only the start position repeated: the car drove a real distance but never
// reported a new place. That is not where it ended, so the end is left unknown until the car reports one.
func staleEnd(t *store.Trip) bool {
	return t.DistanceKm >= 0.8 && t.StartLat != nil && t.StartLon != nil && t.EndLat != nil && t.EndLon != nil &&
		*t.StartLat == *t.EndLat && *t.StartLon == *t.EndLon
}

// settleEnd gives the trip that was just closed with an unknown end the first new position the car reports while parked.
func (d *Detector) settleEnd(st *ford.State) error {
	if d.endlessTrip == 0 {
		return nil
	}
	if st.TS.Unix()-d.endlessAt > 3*3600 || Driving(st) {
		d.endlessTrip = 0
		return nil
	}
	if st.Lat == nil || st.Lon == nil {
		return nil
	}
	t, err := d.S.GetTrip(d.endlessTrip)
	d.endlessTrip = 0
	if err != nil || t == nil || t.EndLat != nil || t.StartLat == nil {
		return err
	}
	if *st.Lat == *t.StartLat && *st.Lon == *t.StartLon {
		return nil // still the old position
	}
	t.EndLat, t.EndLon = st.Lat, st.Lon
	return d.S.UpdateTrip(t)
}
