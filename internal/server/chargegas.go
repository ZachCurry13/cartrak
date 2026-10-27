package server

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/gasprice"
	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/savings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const litersPerGallon = 3.785411784

// chargeGas is what the energy one charge added would have cost in the gas car: it would go as far as the car's recent
// efficiency says, burn that much fuel, at the gas price where the charge happened.
type chargeGas struct {
	Km, Liters, GasCost float64
	Source              string // where the gas price came from
}

type latlon struct{ Lat, Lon float64 }

// gasLiveState holds the live gas price service and the places still waiting for their state.
type gasLiveState struct {
	mu      sync.Mutex
	svc     *gasprice.Service
	pending map[string]latlon
	busy    bool
}

func (s *Server) gasService() *gasprice.Service {
	s.gasLive.mu.Lock()
	defer s.gasLive.mu.Unlock()
	if s.gasLive.svc == nil {
		s.gasLive.svc = &gasprice.Service{KV: s.S, BaseURL: s.GasURL}
	}
	return s.gasLive.svc
}

// geoClient is the shared city lookup client, nil when city lookup is switched off.
func (s *Server) geoClient() *geocode.Client {
	if c, ok := s.places().(*geocode.Client); ok {
		return c
	}
	return nil
}

// liveGasOn says whether prices come from the EIA: a key, US units and city lookup (it names the state) all on.
func (s *Server) liveGasOn() bool {
	return s.Cfg != nil && s.Cfg.EIAKey() != "" && s.Cfg.LiveGasUnits() && s.geoClient() != nil
}

// gasPriceAt is the gas price per liter that applies to a charge and where it came from: the EIA average for the state
// (or its region) that week with the owner's adjustment for the place, else the owner's own price.
func (s *Server) gasPriceAt(c charging.Session, manual float64) (float64, string) {
	own := func() (float64, string) { return manual, "your gas price" }
	if !s.liveGasOn() || c.Lat == nil || c.Lon == nil {
		return own()
	}
	geo := s.geoClient()
	state, ok := geo.CachedState(*c.Lat, *c.Lon)
	if !ok {
		s.queueState(*c.Lat, *c.Lon)
		return own()
	}
	p, ok := s.gasService().At(s.Cfg.EIAKey(), gasprice.ForState(state), c.StartTS)
	if !ok {
		return own()
	}
	price, src := p.PerGallon/litersPerGallon, p.Area
	city, _ := geo.Cached(*c.Lat, *c.Lon)
	for _, a := range s.Cfg.GasAdjusts() {
		if strings.EqualFold(a.Area, city) || strings.EqualFold(a.Area, state) {
			price *= 1 + a.Pct/100
			src = fmt.Sprintf("%s %+g%%", src, a.Pct)
			break
		}
	}
	return price, src
}

// queueState looks up the state of a place in the background, a few at a time, so a page never waits on the lookup service.
func (s *Server) queueState(lat, lon float64) {
	g := &s.gasLive
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending == nil {
		g.pending = map[string]latlon{}
	}
	if len(g.pending) < 40 {
		g.pending[geocode.Key(lat, lon)] = latlon{Lat: lat, Lon: lon}
	}
	if g.busy {
		return
	}
	g.busy = true
	go func() {
		for {
			g.mu.Lock()
			var k string
			var p latlon
			for k, p = range g.pending {
				break
			}
			if k == "" {
				g.busy = false
				g.mu.Unlock()
				return
			}
			delete(g.pending, k)
			g.mu.Unlock()
			if c := s.geoClient(); c != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				_, _ = c.State(ctx, p.Lat, p.Lon)
				cancel()
			}
		}
	}()
}

// efficiencyFor builds the car's distance-per-kWh history from its finished trips.
func (s *Server) efficiencyFor() *savings.Efficiency {
	ts, err := s.S.ListTrips(store.TripFilter{All: true})
	if err != nil {
		return savings.NewEfficiency(nil)
	}
	capacity := s.capacityKwh()
	var ds []savings.Drive
	for _, t := range ts {
		if t.Open || t.EndTS == nil {
			continue
		}
		if o := withEnergy(t, capacity); o.EnergyKwh != nil {
			ds = append(ds, savings.Drive{TS: t.StartTS, Km: t.DistanceKm, Kwh: *o.EnergyKwh})
		}
	}
	return savings.NewEfficiency(ds)
}

// gasEquivalents works out, for every finished charge that added energy, what the same range would have cost in the gas
// car. Charges it cannot judge (no efficiency figure yet) are absent from the result. Nil when no gas car is set.
func (s *Server) gasEquivalents(ss []charging.Session) map[int64]chargeGas {
	if s.Cfg == nil || !s.Cfg.Gas().Set() {
		return nil
	}
	gas, eff := s.Cfg.Gas(), s.efficiencyFor()
	out := map[int64]chargeGas{}
	for _, c := range ss {
		if c.Open || c.AddedKwh == nil || *c.AddedKwh <= 0 {
			continue
		}
		perKwh := eff.KmPerKwh(c.StartTS)
		if perKwh <= 0 {
			continue
		}
		price, src := s.gasPriceAt(c, gas.PricePerLiter)
		km := *c.AddedKwh * perKwh
		liters, cost := savings.Gas(km, gas.LitersPerKm, price)
		out[c.ID] = chargeGas{Km: km, Liters: liters, GasCost: cost, Source: src}
	}
	return out
}

// sourcesOf lists the gas price sources used, most used first.
func sourcesOf(count map[string]int) []string {
	var out []string
	for k := range count {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		return count[out[i]] > count[out[j]] || count[out[i]] == count[out[j]] && out[i] < out[j]
	})
	return out
}
