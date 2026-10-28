package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/chargecurve"
	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// samplePower is the charging power of one stored reading: the onboard charger's figure when it gave one, otherwise
// worked out from the battery's own current and voltage while a cable is connected (fast chargers report no other
// power). Readings stored before CarTrak knew this have a power of zero, so the raw fields are read as well.
func samplePower(x *store.Sample) float64 {
	if x.ChargeKw != nil && *x.ChargeKw > 0.2 {
		return *x.ChargeKw
	}
	p := strings.ToLower(x.Plug)
	if !strings.Contains(p, "connect") || strings.Contains(p, "dis") || strings.Contains(p, "not") {
		return 0
	}
	f := ford.Flatten(x.Extra)
	if len(f) == 0 {
		return 0
	}
	i := ford.Number(f, "xevBatteryIoCurrent")
	v := x.HVVolts
	if v == nil {
		v = ford.Number(f, "xevBatteryVoltage")
	}
	if i == nil || v == nil || *i > -1 {
		return 0
	}
	return -*i * *v / 1000
}

// chargeCurve is the power against battery level for one charging session. GET /api/charges/{id}/curve
func (s *Server) chargeCurve(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	c, err := s.S.GetCharge(id)
	if err != nil || c == nil {
		fail(w, 404, "no such session")
		return
	}
	to := c.StartTS + 12*3600
	if c.EndTS != nil {
		to = *c.EndTS
	}
	rows, err := s.S.SamplesBetween(c.StartTS, to)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var pts []chargecurve.Point
	for _, x := range rows {
		if x.SoCPct == nil {
			continue
		}
		pts = append(pts, chargecurve.Point{TS: x.TS, SoC: *x.SoCPct, Kw: samplePower(x)})
	}
	ok(w, chargecurve.Build(pts, c.StartKwh, c.EndKwh))
}

type chargeOut struct {
	charging.Session
	Place string              `json:"place"`
	Gas   *history.GasCompare `json:"gas,omitempty"` // the same range in the gas car, when a gas car is set
}

// withPlaces adds the name of the named place each session was at.
func (s *Server) withPlaces(ss []charging.Session) []chargeOut {
	all, _ := s.S.ListNamedPlaces()
	out := make([]chargeOut, 0, len(ss))
	for _, x := range ss {
		o := chargeOut{Session: x}
		if len(all) > 0 && x.Lat != nil && x.Lon != nil {
			o.Place = places.Name(x.Lat, x.Lon, all)
		}
		out = append(out, o)
	}
	return out
}

// fastAverage is how fast the car really charges at fast chargers: the energy added over the time, averaged across
// the sessions that reached 40 kW or more. GET /api/charging/fast-average
func (s *Server) fastAverage(w http.ResponseWriter, r *http.Request) {
	ss, err := s.chargeSessions(300)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var sum float64
	n := 0
	for _, x := range ss {
		if x.Open || x.PeakKw < 40 || x.EndTS == nil || x.AddedKwh == nil {
			continue
		}
		hours := float64(*x.EndTS-x.StartTS) / 3600
		if hours < 5.0/60 {
			continue
		}
		sum += *x.AddedKwh / hours
		n++
	}
	out := map[string]any{"sessions": n, "avgKw": nil}
	if n > 0 {
		out["avgKw"] = sum / float64(n)
	}
	ok(w, out)
}

// fastWindow is how long your fast-charging sessions took to go from one battery level to another, for planning a stop
// that charges between the reserve and the target. It uses only sessions that cover the whole window.
// GET /api/charging/window?from=20&to=80
func (s *Server) fastWindow(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.ParseFloat(r.URL.Query().Get("from"), 64)
	to, _ := strconv.ParseFloat(r.URL.Query().Get("to"), 64)
	if from < 1 || to > 100 || to-from < 10 {
		fail(w, 400, "from and to must be battery levels at least 10 apart")
		return
	}
	ss, err := s.chargeSessions(300)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var minutes float64
	n := 0
	for _, x := range ss {
		if x.Open || x.PeakKw < 40 || x.EndTS == nil {
			continue
		}
		rows, err := s.S.SamplesBetween(x.StartTS, *x.EndTS)
		if err != nil {
			continue
		}
		var ta, tb int64
		for _, p := range rows {
			if p.SoCPct == nil {
				continue
			}
			if ta == 0 && *p.SoCPct >= from {
				ta = p.TS
			}
			if tb == 0 && *p.SoCPct >= to {
				tb = p.TS
				break
			}
		}
		if ta == 0 || tb <= ta || tb-ta < 120 {
			continue
		}
		minutes += float64(tb-ta) / 60
		n++
	}
	out := map[string]any{"sessions": n, "minutesPerPct": nil}
	if n > 0 {
		out["minutesPerPct"] = minutes / float64(n) / (to - from)
	}
	ok(w, out)
}
