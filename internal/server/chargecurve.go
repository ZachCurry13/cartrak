package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/chargecurve"
	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/ford"
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
	Place string `json:"place"`
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
