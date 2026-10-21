package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// tripSoFar is a trip away up to now, for the Day view: how far, and what the driving cost per distance.
type tripSoFar struct {
	Label     string   `json:"label"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Base      string   `json:"base"`
	Km        float64  `json:"km"`
	CostPerKm *float64 `json:"costPerKm"`
}

// stretchPrice is what a kWh cost during a stretch, from its priced sessions; otherwise the usual average price.
func (s *Server) stretchPrice(st store.TimeOff) *float64 {
	loc := s.defaultLoc()
	a, e1 := time.ParseInLocation("2006-01-02", st.From, loc)
	b, e2 := time.ParseInLocation("2006-01-02", st.To, loc)
	if e1 == nil && e2 == nil {
		var cost, kwh float64
		if ss, _, err := s.chargeSessionsEff(1000); err == nil {
			for _, c := range ss {
				if c.Open || c.Cost == nil || c.WallKwh == nil || c.StartTS < a.Unix() || c.StartTS >= b.AddDate(0, 0, 1).Unix() {
					continue
				}
				cost += *c.Cost
				kwh += *c.WallKwh
			}
		}
		if kwh > 0 {
			p := cost / kwh
			return &p
		}
	}
	return s.avgPricePerKwh()
}

// dayCosts sets the day's cost per distance from the energy its driving used at what a kWh cost (a charge that fills the battery
// for several days is not billed to one), and on a trip away returns the whole trip so far worked out the same way.
func (s *Server) dayCosts(sum *daySummary, day time.Time, used, usedKm float64, bases *baseFinder) *tripSoFar {
	key := day.Format("2006-01-02")
	st := s.S.StretchOn(key)
	price := s.avgPricePerKwh()
	if st != nil {
		price = s.stretchPrice(*st)
	}
	sum.CostPerKm = 0
	if price != nil && usedKm > 0 {
		sum.CostPerKm = used * *price / usedKm
	}
	b := bases.forDay(key)
	if st == nil || b == nil { // only a trip away, not a mode spent at home
		return nil
	}
	loc := s.defaultLoc()
	from, _ := time.ParseInLocation("2006-01-02", st.From, loc)
	to, _ := time.ParseInLocation("2006-01-02", st.To, loc)
	end := min(to.AddDate(0, 0, 1).Unix(), time.Now().Unix())
	ts, _ := s.S.ListTrips(store.TripFilter{From: from.Unix(), To: end, Limit: 2000})
	capacity := s.capacityKwh()
	out := &tripSoFar{Label: st.Label, From: st.From, To: st.To, Base: b.Name}
	var kwh, km float64
	for _, t := range ts {
		if t.Open {
			continue
		}
		out.Km += t.DistanceKm
		if e, ok := trips.EnergyUsed(t, capacity); ok {
			kwh += e.Kwh
			km += t.DistanceKm
		}
	}
	if price != nil && km > 0 {
		c := kwh * *price / km
		out.CostPerKm = &c
	}
	return out
}

// markBase names the base as the start or end of trips in a list, for trips taken on a trip away.
func (s *Server) markBase(ts []tripOut) {
	bases := s.baseFinder(nil)
	for i := range ts {
		t := ts[i].Trip
		b := bases.forDay(time.Unix(t.StartTS, 0).In(bases.loc).Format("2006-01-02"))
		if b == nil {
			continue
		}
		atStart, atEnd := b.near(t.StartLat, t.StartLon), b.near(t.EndLat, t.EndLon) && !t.Open
		if atStart && !atEnd && ts[i].StartPlace == "" {
			ts[i].StartPlace = "Base"
		}
		if atEnd && !atStart && ts[i].EndPlace == "" {
			ts[i].EndPlace = "Base"
		}
	}
}

// setTimeOffBase chooses where you are staying for a stretch: {"baseId": 3} for a named place, or {"baseId": null} to let
// CarTrak find it from where the car spends the nights. Admin only.
func (s *Server) setTimeOffBase(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		BaseID *int64 `json:"baseId"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if in.BaseID != nil {
		if p, err := s.S.GetNamedPlace(*in.BaseID); err != nil || p == nil {
			fail(w, 400, "choose one of your places")
			return
		}
	}
	if err := s.S.SetTimeOffBase(id, in.BaseID); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "saved")
}
