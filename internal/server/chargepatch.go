package server

import (
	"net/http"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// patchCharge sets whether a session was at home or public. Marking one as home also tells CarTrak where
// home is, so later sessions there are recognized on their own. {"kind":"home"|"public"|""}. Admin only.
func (s *Server) patchCharge(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Kind         *string  `json:"kind"`
		ChargerKwh   *float64 `json:"chargerKwh"`
		ClearCharger bool     `json:"clearCharger"`
		Cost         *float64 `json:"cost"` // what was paid, from the receipt
		ClearCost    bool     `json:"clearCost"`
	}
	if err := decode(r, &in); err != nil || (in.Kind != nil && *in.Kind != "" && *in.Kind != "home" && *in.Kind != "public") {
		fail(w, 400, "choose home, public or automatic")
		return
	}
	c, err := s.S.GetCharge(id)
	if err != nil || c == nil {
		fail(w, 404, "no such charging session")
		return
	}
	if in.Kind != nil {
		if err := s.S.SetChargeKind(id, *in.Kind); err != nil {
			fail(w, 500, "db error")
			return
		}
		if *in.Kind == "home" && c.Lat != nil && c.Lon != nil {
			_ = s.S.SetKV(kvHomeLat, strconv.FormatFloat(*c.Lat, 'f', 5, 64))
			_ = s.S.SetKV(kvHomeLon, strconv.FormatFloat(*c.Lon, 'f', 5, 64))
		}
	}
	if in.ClearCost {
		_ = s.S.SetChargeCost(id, nil)
	} else if in.Cost != nil {
		if *in.Cost < 0 || *in.Cost > 2000 {
			fail(w, 400, "enter what you paid, for example 13.06")
			return
		}
		if err := s.S.SetChargeCost(id, in.Cost); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	if in.ClearCharger {
		_ = s.S.SetChargerKwh(id, nil)
	} else if in.ChargerKwh != nil {
		added := charging.Enrich(c, s.capacityKwh(), nil, charging.Prices{}).AddedKwh
		switch {
		case *in.ChargerKwh <= 0 || *in.ChargerKwh > 300:
			fail(w, 400, "enter the kWh the charger reported, for example 31.4")
			return
		case added != nil && *in.ChargerKwh < *added*0.95:
			fail(w, 400, "that is less than the energy the battery received, so check the number")
			return
		}
		if err := s.S.SetChargerKwh(id, in.ChargerKwh); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	ok(w, "saved")
}
