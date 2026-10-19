package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/insights"
	"github.com/ZachCurry13/cartrak/internal/speedeff"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tripwhy"
)

var (
	whyMu   sync.Mutex
	whyAt   time.Time
	whyBase tripwhy.Baseline
)

// whyBaseline is what the driver's own history says about speed and temperature, kept for half an hour.
func (s *Server) whyBaseline() tripwhy.Baseline {
	whyMu.Lock()
	defer whyMu.Unlock()
	if time.Since(whyAt) < 30*time.Minute && whyBase.Overall > 0 {
		return whyBase
	}
	var b tripwhy.Baseline
	b.TempKmPKwh = map[string]float64{}
	if rows, err := s.S.DrivingReadings(time.Now().AddDate(0, 0, -90).Unix()); err == nil {
		rs := make([]speedeff.Reading, 0, len(rows))
		for _, x := range rows {
			rs = append(rs, speedeff.Reading{TS: int64(x[0]), Odo: x[1], Kwh: x[2], Speed: x[3]})
		}
		var km, kwh float64
		for _, o := range speedeff.Build(rs) {
			b.Bands = append(b.Bands, tripwhy.Band{LoKph: o.LoKph, HiKph: o.HiKph, KmPerKwh: o.KmPerKwh})
			km += o.Km
			kwh += o.Kwh
		}
		if kwh > 0 {
			b.Overall = km / kwh
		}
	}
	from := time.Now().AddDate(0, 0, -366).Unix()
	if ts, err := s.S.EnergyTrips(from); err == nil {
		if temps, err := s.S.TripTemps(from); err == nil {
			rep := insights.Build(ts, s.capacityKwh(), temps, s.defaultLoc())
			var km, kwh float64
			for _, bd := range rep.Bands {
				b.TempKmPKwh[bd.Name] = bd.KmPerKwh
				km += bd.Km
				kwh += bd.Km / bd.KmPerKwh
			}
			if kwh > 0 {
				b.TempOverall = km / kwh
			}
		}
	}
	whyBase, whyAt = b, time.Now()
	return b
}

// tripWhy explains one trip's efficiency: speed, temperature, stopping and what is left over.
// GET /api/trips/{id}/why
func (s *Server) tripWhy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	e, has := trips.EnergyUsed(t, s.capacityKwh())
	rows, err := s.S.TripReadings(t.StartTS, *t.EndTS)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	rs := make([]tripwhy.Reading, 0, len(rows))
	var tsum float64
	tn := 0
	for _, x := range rows {
		if x.Odo == nil || x.Kwh == nil || x.Speed == nil {
			continue
		}
		rs = append(rs, tripwhy.Reading{TS: x.TS, Odo: *x.Odo, Kwh: *x.Kwh, Speed: *x.Speed, OutsideC: x.Temp})
		if x.Temp != nil {
			tsum += *x.Temp
			tn++
		}
	}
	group := ""
	if tn > 0 {
		group = insights.BandName(tsum / float64(tn))
	}
	kwh := 0.0
	if has {
		kwh = e.Kwh
	}
	// The car keeps its own count of the range braking gave back since the ignition came on; its last value in the trip
	// is the trip's total. (Its other regen counter is documented as a percentage but reads like a different unit.)
	var regen *float64
	if x, _ := s.S.SampleAt(*t.EndTS); x != nil && x.TS >= t.StartTS {
		f := map[string]any{}
		if json.Unmarshal(x.Extra, &f) == nil {
			regen = ford.Number(f, "tripXevBatteryRangeRegenerated")
		}
	}
	ok(w, tripwhy.Build(rs, kwh, s.whyBaseline(), group, regen))
}
