// Package charging turns recorded charging sessions into energy added and a cost.
package charging

import (
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	homeRadiusM  = 150.0
	acKwLimit    = 20.0 // a session that never passed this power was on a slow (AC) charger
	acEfficiency = 0.9  // about a tenth of the energy from the wall is lost as heat while charging on AC
	minAddedKwh  = 0.1
)

// Point is a place on the map.
type Point struct{ Lat, Lon float64 }

// Prices are per kWh; nil means not set.
type Prices struct {
	Home, Public *float64
	OffPeak      *float64       // cheaper home price inside Window, when the plan has one
	Window       *Window        // when OffPeak applies
	Loc          *time.Location // the zone Window is read in
	AcEfficiency float64        // share of the wall energy that reaches the battery on slow charging; 0 uses the default
	Places       []PlaceRate    // named places with their own charging price or that count as home
}

// PlaceRate is a named place that changes how charging there is priced or counted.
type PlaceRate struct {
	Name     string
	Lat, Lon float64
	RadiusM  float64
	Price    *float64 // per kWh; nil means the usual price
	Home     bool
}

// placeAt is the named place that holds a position, the nearest if several do.
func (p Prices) placeAt(lat, lon float64) *PlaceRate {
	var best *PlaceRate
	bestD := 0.0
	for i := range p.Places {
		pl := &p.Places[i]
		d := distanceM(Point{pl.Lat, pl.Lon}, Point{lat, lon})
		if d <= pl.RadiusM && (best == nil || d < bestD) {
			best, bestD = pl, d
		}
	}
	return best
}

// Session is a charging session with what it added and cost.
type Session struct {
	*store.Charge
	AddedKwh   *float64 `json:"addedKwh"`  // energy that went into the battery
	Estimated  bool     `json:"estimated"` // worked out from the battery percent, not the car's energy reading
	WallKwh    *float64 `json:"wallKwh"`   // what the charger delivered, allowing for AC losses
	Where      string   `json:"where"`     // home, public, or "" when it cannot be told
	Place      string   `json:"place"`     // the named place it was at, when that place sets its own price or counts as home
	Cost       *float64 `json:"cost"`
	CostActual bool     `json:"costActual"`       // the cost is what the owner paid, not worked out from a price
	PriceFrom  string   `json:"priceFrom"`        // where the cost came from: receipt, place, home, home+night, public, or "" when unpriced
	Efficiency *float64 `json:"efficiency"`       // added / delivered, only when the charger's number was entered
	Remote     bool     `json:"remote,omitempty"` // a remote start with the car plugged in, not a charge (set by the server)
	Measured   bool     `json:"measured"`
}

// DistanceM is the distance in meters between two points.
func DistanceM(a, b Point) float64 { return distanceM(a, b) }

func distanceM(a, b Point) float64 {
	const r = 6371000.0
	la1, la2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dLat, dLon := (b.Lat-a.Lat)*math.Pi/180, (b.Lon-a.Lon)*math.Pi/180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// Enrich adds energy, place and cost to a finished or running session. capacityKwh and home may be nil.
func Enrich(c *store.Charge, capacityKwh *float64, home *Point, p Prices) Session {
	s := enrichBase(c, capacityKwh, home, p)
	if c.ActualCost != nil { // what was really paid beats any price per kWh
		v := *c.ActualCost
		s.Cost, s.CostActual, s.PriceFrom = &v, true, "receipt"
	}
	return s
}

func enrichBase(c *store.Charge, capacityKwh *float64, home *Point, p Prices) Session {
	s := Session{Charge: c, Where: c.Kind}
	switch {
	case c.StartKwh != nil && c.EndKwh != nil:
		if d := *c.EndKwh - *c.StartKwh; d >= minAddedKwh {
			s.AddedKwh = &d
		}
	case c.StartSoC != nil && c.EndSoC != nil && capacityKwh != nil && *capacityKwh > 0:
		if d := (*c.EndSoC - *c.StartSoC) / 100 * *capacityKwh; d >= minAddedKwh {
			s.AddedKwh, s.Estimated = &d, true
		}
	}
	if s.Where == "" && home != nil && c.Lat != nil && c.Lon != nil {
		s.Where = "public"
		if distanceM(*home, Point{*c.Lat, *c.Lon}) <= homeRadiusM {
			s.Where = "home"
		}
	}
	if s.AddedKwh == nil {
		return s
	}
	wall := *s.AddedKwh
	switch {
	case c.ChargerKwh != nil && *c.ChargerKwh >= wall*0.95:
		wall = *c.ChargerKwh
		eff := *s.AddedKwh / wall
		s.Efficiency, s.Measured = &eff, true
	case c.PeakKw < acKwLimit:
		eff := p.AcEfficiency
		if eff < 0.5 || eff > 1 {
			eff = acEfficiency
		}
		wall /= eff
		// The battery rose by less than the charger delivered when the car also warmed itself from the wall (a top-up
		// before leaving, cold weather). The power the car reported while plugged in sees that.
		if c.PowerKwh != nil && *c.PowerKwh/eff > wall {
			wall = *c.PowerKwh / eff
		}
	}
	s.WallKwh = &wall
	var rate *PlaceRate // a named place can set its own price, or count as home
	if c.Kind == "" && c.Lat != nil && c.Lon != nil {
		if rate = p.placeAt(*c.Lat, *c.Lon); rate != nil && (rate.Price != nil || rate.Home) {
			s.Place = rate.Name
			if rate.Home {
				s.Where = "home"
			} else if s.Where == "" || s.Where == "home" {
				s.Where = "public"
			}
		} else {
			rate = nil
		}
	}
	price, from := p.Home, "home" // unknown place: the home price, the usual single-price case
	if rate != nil && rate.Price != nil {
		price, from = rate.Price, "place"
	} else if s.Where == "public" {
		price, from = p.Public, "public"
	} else if price != nil && p.OffPeak != nil && p.Window != nil && p.Loc != nil {
		from = "home+night"
		end := c.StartTS
		if c.EndTS != nil {
			end = *c.EndTS
		}
		share := p.Window.Share(c.StartTS, end, p.Loc) // the part of the session in the cheap hours
		blended := share**p.OffPeak + (1-share)**price
		price = &blended
	}
	if price != nil {
		cost := wall * *price
		s.Cost, s.PriceFrom = &cost, from
	}
	return s
}
