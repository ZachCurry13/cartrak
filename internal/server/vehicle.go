package server

import (
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

// vehicleInfo says what kind of car this is and which parts of the app apply to it.
type vehicleInfo struct {
	Powertrain string `json:"powertrain"` // ev | phev | hev | gas, or empty until known
	Label      string `json:"label"`
	Source     string `json:"source"` // setting | garage | detected | unknown
	Battery    bool   `json:"battery"`
	Plug       bool   `json:"plug"`
	Engine     bool   `json:"engine"`
}

// powertrain is the admin's choice if there is one, then the engine type in Ford's garage
// record, then what the car has been seen to report.
func (s *Server) powertrain() (ford.Powertrain, string) {
	if s.Cfg != nil {
		if p := ford.Powertrain(s.Cfg.Value(settings.PowertrainKey)); p.Valid() {
			return p, "setting"
		}
	}
	if p := ford.FromEngineType(s.S.GetKV(ford.EngineKV)); p != ford.Unknown {
		return p, "garage"
	}
	if p := ford.ParseCaps(s.S.GetKV(ford.CapsKV)).Powertrain(); p != ford.Unknown {
		return p, "detected"
	}
	return ford.Unknown, "unknown"
}

func (s *Server) vehicleInfo() vehicleInfo {
	p, src := s.powertrain()
	return vehicleInfo{string(p), p.Label(), src, p.HasBattery(), p.CanPlug(), p.HasEngine()}
}

func (s *Server) vehicle(w http.ResponseWriter, r *http.Request) { ok(w, s.vehicleInfo()) }
