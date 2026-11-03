package server

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/chargeeta"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// etaHistory keeps what was learned from the past charging sessions for a few minutes, so asking for an estimate every half minute
// does not read every old session again.
type etaHistory struct {
	mu       sync.Mutex
	at       time.Time
	sessions []chargeeta.Session
}

const (
	etaKeep     = 10 * time.Minute
	etaSessions = 40 // how many recent finished sessions it learns from
)

// sessionFrom turns the readings of one finished charge into what the estimate learns from: the kind of charger, the battery and
// outside temperature at the start, and power against battery level.
func sessionFrom(rows []*store.Sample) chargeeta.Session {
	var s chargeeta.Session
	var typ string
	var kws []float64
	for _, x := range rows {
		if x.SoCPct == nil {
			continue
		}
		kw := samplePower(x)
		s.Obs = append(s.Obs, chargeeta.Obs{SoC: *x.SoCPct, Kw: kw})
		if kw > 0.3 {
			kws = append(kws, kw)
		}
		f := ford.Flatten(x.Extra)
		if typ == "" {
			typ = ford.Text(f, "xevChargeStationPowerType")
		}
		if s.BatC == nil {
			s.BatC = ford.Number(f, "xevBatteryTemperature")
		}
		if s.OutC == nil {
			s.OutC = x.OutsideC
		}
	}
	sort.Float64s(kws)
	var mid float64
	if len(kws) > 0 {
		mid = kws[len(kws)/2]
	}
	s.Level = chargeeta.LevelOf(typ, mid)
	return s
}

func (s *Server) etaSessions() []chargeeta.Session {
	h := &s.etaHist
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessions != nil && time.Since(h.at) < etaKeep {
		return h.sessions
	}
	out := []chargeeta.Session{}
	cs, _ := s.S.ListCharges(etaSessions * 2)
	for _, c := range cs {
		if c.Open || c.EndTS == nil || len(out) >= etaSessions {
			continue
		}
		rows, err := s.S.SamplesBetween(c.StartTS, *c.EndTS)
		if err != nil {
			continue
		}
		if sess := sessionFrom(rows); len(sess.Obs) >= 4 {
			out = append(out, sess)
		}
	}
	h.sessions, h.at = out, time.Now()
	return out
}

// chargeETA is how long the charge in progress will take to reach a level, learned from this car's own earlier sessions of the same
// kind of charger and battery temperature. GET /api/charges/eta?target=90
func (s *Server) chargeETA(w http.ResponseWriter, r *http.Request) {
	target, _ := strconv.Atoi(r.URL.Query().Get("target"))
	if target < 20 || target > 100 {
		target = 80
	}
	x, err := s.S.LatestSample()
	if err != nil || x == nil || x.SoCPct == nil {
		ok(w, map[string]any{"idle": true})
		return
	}
	kw := samplePower(x)
	if kw < 0.3 {
		ok(w, map[string]any{"idle": true})
		return
	}
	if x.CapacityKwh == nil || *x.CapacityKwh <= 0 {
		ok(w, map[string]any{"unavailable": true})
		return
	}
	f := ford.Flatten(x.Extra)
	bat := ford.Number(f, "xevBatteryTemperature")
	now := chargeeta.Now{Level: chargeeta.LevelOf(ford.Text(f, "xevChargeStationPowerType"), kw), SoC: *x.SoCPct, Kw: kw, BatC: bat, OutC: x.OutsideC}
	res, found := chargeeta.Estimate(s.etaSessions(), now, float64(target), *x.CapacityKwh)
	if !found {
		ok(w, map[string]any{"done": float64(target) <= *x.SoCPct})
		return
	}
	ok(w, map[string]any{"minutes": res.Minutes, "learned": res.Learned, "sessions": res.Sessions, "level": res.Level, "tempClass": res.TempClass,
		"preconditioned": res.Preconditioned, "batteryC": bat, "outsideC": x.OutsideC, "kw": kw})
}
