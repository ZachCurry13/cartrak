package trips

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tzlookup"
)

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
