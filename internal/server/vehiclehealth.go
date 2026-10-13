package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/vhealth"
)

// vehicleHealth is the data behind the Vehicle health tab: the latest reading arranged by what
// matters for this type of car, plus a few daily trends. It never includes the VIN or location.
func (s *Server) vehicleHealth(w http.ResponseWriter, r *http.Request) {
	x, err := s.S.LatestSample()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	info := s.vehicleInfo()
	if x == nil {
		ok(w, map[string]any{"vehicle": info, "ready": false})
		return
	}
	f := map[string]any{}
	_ = json.Unmarshal(x.Extra, &f)
	now := time.Now()
	trends := vhealth.Trends{}
	for name, days := range map[string]int{"battery12v": 90, "capacityKwh": 365, "oilLifePct": 365} {
		if d, err := s.S.DailySeries(name, now.AddDate(0, 0, -days).Unix()); err == nil {
			trends[name] = d
		}
	}
	ok(w, map[string]any{
		"vehicle": info,
		"ready":   true,
		"asOf":    x.TS,
		"model":   modelName(f),
		"report":  vhealth.Build(f, ford.Powertrain(info.Powertrain), trends),
	})
}

// modelName is for example "2024 Mustang Mach-E", from Ford's garage record.
func modelName(f map[string]any) string {
	var parts []string
	for _, k := range []string{"garage.modelYear", "garage.modelName"} {
		if v := ford.Text(f, k); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}
