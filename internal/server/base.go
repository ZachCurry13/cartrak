package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// baseRadiusM is how close a place must be to the base to count as being at it: a hotel car park or a street near a rental.
const baseRadiusM = 300.0

// tripBase is where you are staying on a day of a trip away: the place the car spends the nights, or the named place chosen for
// the stretch. Time parked there is not a stop, and drives start from and return to it.
type tripBase struct {
	Lat, Lon float64
	Name     string // a named place or the town, "" when neither is known yet
	Stretch  store.TimeOff
}

// near says whether a point is at the base.
func (b *tripBase) near(lat, lon *float64) bool {
	return b != nil && lat != nil && lon != nil && charging.DistanceM(charging.Point{Lat: b.Lat, Lon: b.Lon}, charging.Point{Lat: *lat, Lon: *lon}) <= baseRadiusM
}

// label is how the base is named in a row: "Base", with the place when it is known.
func (b *tripBase) label() string {
	if b.Name == "" {
		return "Base"
	}
	return "Base · " + b.Name
}

// baseFinder works out the base of each day once per request.
type baseFinder struct {
	s      *Server
	loc    *time.Location
	named  []*store.NamedPlace
	home   *charging.Point
	cities *cityNamer
	memo   map[string]*tripBase
}

func (s *Server) baseFinder(cities *cityNamer) *baseFinder {
	named, _ := s.S.ListNamedPlaces()
	return &baseFinder{s: s, loc: s.defaultLoc(), named: named, home: s.homePoint(), cities: cities, memo: map[string]*tripBase{}}
}

// night is where the car was at three in the morning of a day, from its last report in the hours before.
func (f *baseFinder) night(day time.Time) (float64, float64, bool) {
	at := time.Date(day.Year(), day.Month(), day.Day(), 3, 0, 0, 0, f.loc)
	if at.After(time.Now()) {
		return 0, 0, false
	}
	return f.s.S.PositionAt(at.Unix(), 12*3600)
}

// forDay is the base on a day (YYYY-MM-DD), or nil when the day is not in a stretch or the car sleeps at home. The night before a
// day counts, except on the first day of a stretch, when it is the night after (the night before was still at home).
func (f *baseFinder) forDay(day string) *tripBase {
	if b, seen := f.memo[day]; seen {
		return b
	}
	b := f.find(day)
	f.memo[day] = b
	return b
}

func (f *baseFinder) find(day string) *tripBase {
	st := f.s.S.StretchOn(day)
	if st == nil {
		return nil
	}
	if st.BaseID != nil { // the owner chose the place
		for _, p := range f.named {
			if p.ID == *st.BaseID {
				return &tripBase{Lat: p.Lat, Lon: p.Lon, Name: p.Name, Stretch: *st}
			}
		}
	}
	d, err := time.ParseInLocation("2006-01-02", day, f.loc)
	if err != nil {
		return nil
	}
	lat, lon, ok := 0.0, 0.0, false
	if day > st.From {
		lat, lon, ok = f.night(d)
	}
	if !ok && day < st.To {
		lat, lon, ok = f.night(d.AddDate(0, 0, 1))
	}
	if !ok {
		return nil
	}
	if f.home != nil && charging.DistanceM(*f.home, charging.Point{Lat: lat, Lon: lon}) <= baseRadiusM {
		return nil // the car sleeps at home: not away
	}
	name := places.Name(&lat, &lon, f.named)
	if name == "" && f.cities != nil {
		name = f.cities.name(&lat, &lon)
	}
	return &tripBase{Lat: lat, Lon: lon, Name: name, Stretch: *st}
}
