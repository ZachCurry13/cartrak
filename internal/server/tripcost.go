package server

import (
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

// chargePrices is what a kWh has cost at each kind of charger, from the owner's own charges that have a cost, so a Tesla or
// Electrify America discount or subscription is already in it. A network with fewer than two such charges is left out and the
// public average (every charge away from home) stands in for it.
type chargePrices struct {
	Tesla  *float64 `json:"tesla,omitempty"`
	EA     *float64 `json:"ea,omitempty"`
	Public *float64 `json:"public,omitempty"`
}

func (s *Server) learnedPrices() chargePrices {
	type acc struct {
		cost, kwh float64
		n         int
	}
	var tesla, ea, public acc
	ss, err := s.chargeSessions(1000)
	if err != nil {
		return chargePrices{}
	}
	for _, c := range ss {
		if c.Open || c.Remote || c.Cost == nil || c.AddedKwh == nil || *c.AddedKwh < 1 || c.Where == "home" {
			continue
		}
		add := func(a *acc) { a.cost, a.kwh, a.n = a.cost+*c.Cost, a.kwh+*c.AddedKwh, a.n+1 }
		add(&public)
		switch name := s.chargerAt(c.Lat, c.Lon); {
		case strings.HasPrefix(name, "Supercharger:"):
			add(&tesla)
		case strings.HasPrefix(name, "Electrify America:"):
			add(&ea)
		}
	}
	per := func(a acc, min int) *float64 {
		if a.n < min || a.kwh <= 0 {
			return nil
		}
		v := a.cost / a.kwh
		return &v
	}
	return chargePrices{Tesla: per(tesla, 2), EA: per(ea, 2), Public: per(public, 2)}
}

// fillTimesAndCosts gives each day its start time (the same time of day on each day of the trip) and each charge its expected cost.
func (s *Server) fillTimesAndCosts(days []tripchain.Day, departTS int64) chargePrices {
	if departTS <= 0 {
		t := time.Now().In(s.defaultLoc())
		departTS = time.Date(t.Year(), t.Month(), t.Day()+1, 8, 0, 0, 0, t.Location()).Unix()
	}
	p := s.learnedPrices()
	for i := range days {
		days[i].DepartTS = time.Unix(departTS, 0).AddDate(0, 0, i).Unix()
		for j := range days[i].Stops {
			st := &days[i].Stops[j]
			if st.Kwh <= 0 {
				continue
			}
			price := p.Public
			switch {
			case st.Net != "":
				if p.EA != nil {
					price = p.EA
				}
			case p.Tesla != nil:
				price = p.Tesla
			}
			if price != nil {
				c := st.Kwh * *price
				st.Cost = &c
			}
		}
	}
	return p
}
