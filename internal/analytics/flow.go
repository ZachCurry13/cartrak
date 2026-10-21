package analytics

import (
	"sort"
	"time"
)

// Flow is where the energy charged in a stretch of time went.
type Flow struct {
	HomeKwh    float64 `json:"homeKwh"`    // from the charger at home
	PublicKwh  float64 `json:"publicKwh"`  // from public chargers
	LostKwh    float64 `json:"lostKwh"`    // lost between the charger and the battery
	CityKwh    float64 `json:"cityKwh"`    // driving at under 70 km/h on average
	HighwayKwh float64 `json:"highwayKwh"` // driving at 70 km/h or more on average
	OtherKwh   float64 `json:"otherKwh"`   // climate and keeping the car awake while parked, and the rest
	StoredKwh  float64 `json:"storedKwh"`  // left in the battery (negative when it ended lower than it began)
	CityKm     float64 `json:"cityKm"`
	HighwayKm  float64 `json:"highwayKm"`
}

const highwayKph = 70

// EnergyFlow follows the energy charged between from and to. storedKwh is how much more is in the battery at the end than at
// the start; with it the rest of what was charged and not driven is the car's own use while parked or warming.
func EnergyFlow(ts []Trip, cs []Charge, from, to int64, storedKwh float64) Flow {
	f := Flow{StoredKwh: storedKwh}
	var added float64
	for _, c := range cs {
		if c.Start < from || c.Start >= to || c.Kwh <= 0 {
			continue
		}
		wall := c.WallKwh
		if wall < c.Kwh {
			wall = c.Kwh
		}
		if c.Where == "home" {
			f.HomeKwh += wall
		} else {
			f.PublicKwh += wall
		}
		f.LostKwh += wall - c.Kwh
		added += c.Kwh
	}
	for _, t := range ts {
		if t.Start < from || t.Start >= to || t.Kwh <= 0 || t.End <= t.Start {
			continue
		}
		if t.Km/(float64(t.End-t.Start)/3600) >= highwayKph {
			f.HighwayKwh += t.Kwh
			f.HighwayKm += t.Km
		} else {
			f.CityKwh += t.Kwh
			f.CityKm += t.Km
		}
	}
	f.OtherKwh = max(0, added-f.CityKwh-f.HighwayKwh-storedKwh)
	return f
}

// DayRing is a typical day: for each quarter hour, the share of days the car was driving or charging then.
type DayRing struct {
	Driving  [96]float64 `json:"driving"`
	Charging [96]float64 `json:"charging"`
	Parked   float64     `json:"parked"` // share of all the time it was neither
	Days     int         `json:"days"`
	LeaveAt  string      `json:"leaveAt"`  // the usual first departure on a weekday, "07:15", or ""
	PlugInAt string      `json:"plugInAt"` // the usual time the evening charge starts, or ""
}

// TypicalDay builds the ring over the days from..to.
func TypicalDay(ts []Trip, cs []Charge, from, to int64, loc *time.Location) DayRing {
	var r DayRing
	// count only from the first day there is anything: a year that began before CarTrak did is not a year of the car parked
	first := to
	for _, t := range ts {
		first = min(first, t.Start)
	}
	for _, c := range cs {
		first = min(first, c.Start)
	}
	if first > from && first < to {
		y, m, d := time.Unix(first, 0).In(loc).Date()
		from = max(from, time.Date(y, m, d, 0, 0, 0, 0, loc).Unix())
	}
	days := float64(to-from) / 86400
	if days < 1 {
		return r
	}
	r.Days = int(days + 0.5)
	mark := func(arr *[96]float64, a, b int64) float64 {
		a, b = max(a, from), min(b, to)
		for t := a - a%900; t < b; t += 900 {
			s := time.Unix(t, 0).In(loc)
			arr[(s.Hour()*60+s.Minute())/15] += 1 / days
		}
		return float64(max(0, b-a))
	}
	var busy float64
	var leaves, plugs []int
	firstOfDay := map[string]int{}
	for _, t := range ts {
		busy += mark(&r.Driving, t.Start, t.End)
		st := time.Unix(t.Start, 0).In(loc)
		if st.Weekday() == time.Saturday || st.Weekday() == time.Sunday {
			continue
		}
		d, m := st.Format("2006-01-02"), st.Hour()*60+st.Minute()
		if v, ok := firstOfDay[d]; !ok || m < v {
			firstOfDay[d] = m
		}
	}
	for _, c := range cs {
		busy += mark(&r.Charging, c.Start, c.End)
		if hr := time.Unix(c.Start, 0).In(loc).Hour(); hr >= 16 || hr < 3 {
			st := time.Unix(c.Start, 0).In(loc)
			m := st.Hour()*60 + st.Minute()
			if hr < 3 {
				m += 24 * 60
			}
			plugs = append(plugs, m)
		}
	}
	for _, v := range firstOfDay {
		leaves = append(leaves, v)
	}
	for i := range r.Driving {
		r.Driving[i], r.Charging[i] = min(1, r.Driving[i]), min(1, r.Charging[i])
	}
	r.Parked = max(0, 1-busy/float64(to-from))
	r.LeaveAt, r.PlugInAt = clockOf(median(leaves)), clockOf(median(plugs))
	return r
}

func median(v []int) int {
	if len(v) < 3 {
		return -1
	}
	s := append([]int(nil), v...)
	sort.Ints(s)
	return s[len(s)/2]
}

func clockOf(m int) string {
	if m < 0 {
		return ""
	}
	m %= 24 * 60
	m = (m + 2) / 5 * 5 % (24 * 60)
	return time.Date(2000, 1, 1, m/60, m%60, 0, 0, time.UTC).Format("15:04")
}
