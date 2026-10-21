package server

import (
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// When the Ford account linked to this car authorized more than one vehicle (a family with an F-150 and an Escape on one
// account, say), the owner chooses which one this car follows. Each vehicle another car already follows is named.

// CarRef names a car.
type CarRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type vehicleLister interface{ Vehicles() []ford.VehicleChoice }

type vehicleOwners interface {
	VehicleOwners() map[string][]CarRef
}

// fordVehicles: GET /api/ford/vehicles. The authorized vehicles, which one this car follows, and which other car follows each.
func (s *Server) fordVehicles(w http.ResponseWriter, r *http.Request) {
	l, ok2 := s.Link.(vehicleLister)
	if !ok2 {
		ok(w, map[string]any{"vehicles": []any{}})
		return
	}
	following := strings.ToUpper(s.S.GetKV(ford.SeenVINKey))
	var owners map[string][]CarRef
	if o, ok3 := s.Cars.(vehicleOwners); ok3 && s.Cars != nil {
		owners = o.VehicleOwners()
	}
	type row struct {
		VIN    string   `json:"vin"`
		Label  string   `json:"label"`
		This   bool     `json:"this"`
		UsedBy []string `json:"usedBy"`
	}
	out := []row{}
	for _, v := range l.Vehicles() {
		rw := row{VIN: v.VIN, Label: v.Label, This: strings.EqualFold(v.VIN, following), UsedBy: []string{}}
		for _, c := range owners[strings.ToUpper(v.VIN)] {
			if c.ID != s.CarID {
				rw.UsedBy = append(rw.UsedBy, c.Name)
			}
		}
		out = append(out, rw)
	}
	ok(w, map[string]any{"vehicles": out})
}

// chooseVehicle: POST /api/ford/vehicle {"vin": "..."}. This car follows that vehicle from the next reading on.
func (s *Server) chooseVehicle(w http.ResponseWriter, r *http.Request) {
	var in struct {
		VIN string `json:"vin"`
	}
	l, ok2 := s.Link.(vehicleLister)
	if decode(r, &in) != nil || !ok2 || s.Cfg == nil {
		fail(w, 400, "bad request")
		return
	}
	for _, v := range l.Vehicles() {
		if strings.EqualFold(v.VIN, in.VIN) {
			if err := s.Cfg.Set("FORD_VIN", v.VIN); err != nil {
				fail(w, 500, "could not save")
				return
			}
			_ = s.S.SetKV(ford.SeenVINKey, v.VIN)
			ok(w, map[string]string{"label": v.Label})
			return
		}
	}
	fail(w, 404, "that vehicle is not on the linked Ford account")
}
