package trips

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// The owner can say what the outing in progress is, before leaving Home or part-way through: "This outing is work". The trips it
// has so far are relabeled by hand, and the ones that follow, until the car is back at Home, take the same category.
const (
	kvOuting      = "outing_override" // "category|unix time it was set"
	outingKeepSec = 18 * 3600         // a forgotten setting lapses on its own
)

// outingOverride is the category set for the outing under way at startTS, or "".
func (d *Detector) outingOverride(startTS int64) string {
	f := strings.SplitN(d.S.GetKV(kvOuting), "|", 2)
	if len(f) != 2 {
		return ""
	}
	setAt, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil || startTS < setAt || startTS > setAt+outingKeepSec || !d.S.CategoryExists(f[0]) {
		return ""
	}
	return f[0]
}

// Outing is the category the owner set for the outing in progress ("" when none) and how many trips the outing has so far.
func (d *Detector) Outing(now int64) (cat string, trips int) {
	cat = d.outingOverride(now)
	return cat, len(d.currentOuting(now))
}

// SetOuting sets the category of the outing in progress, or clears the setting when cat is empty. It returns how many trips were
// relabeled.
func (d *Detector) SetOuting(cat string, now int64) (int, error) {
	if cat == "" {
		return 0, d.S.SetKV(kvOuting, "")
	}
	if !d.S.CategoryExists(cat) {
		return 0, fmt.Errorf("there is no category %q", cat)
	}
	ts := d.currentOuting(now)
	for _, t := range ts {
		if err := d.S.SetTripCategory(t.ID, cat, t.Notes); err != nil {
			return 0, err
		}
	}
	return len(ts), d.S.SetKV(kvOuting, fmt.Sprintf("%s|%d", cat, now))
}

// currentOuting is the trips of the outing in progress, newest first: back from the latest trip to the one that left Home, unless
// the car is already home, when there is none.
func (d *Detector) currentOuting(now int64) []*store.Trip {
	ps, err := d.S.ListTrips(store.TripFilter{From: now - outingGapSec, To: now + 1, All: true})
	if err != nil {
		return nil
	}
	places, _ := d.S.ListNamedPlaces()
	spots := homeSpots(places)
	var out []*store.Trip
	for _, t := range ps {
		if _, home := atHome(spots, t.EndLat, t.EndLon); home && !t.Open {
			break // it ended at Home: the outing began after it
		}
		out = append(out, t)
		if _, home := atHome(spots, t.StartLat, t.StartLon); home {
			break // this one left Home
		}
	}
	return out
}

// endOuting forgets the owner's setting once a trip has ended at Home.
func (d *Detector) endOuting(t *store.Trip) {
	if d.S.GetKV(kvOuting) == "" || t.Open {
		return
	}
	places, _ := d.S.ListNamedPlaces()
	if _, home := atHome(homeSpots(places), t.EndLat, t.EndLon); home {
		_ = d.S.SetKV(kvOuting, "")
	}
}
