package server

import (
	"net/http"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/insights"
)

type rangeWindow struct {
	From    float64  `json:"from"`
	To      float64  `json:"to"`
	Km      float64  `json:"km"`
	Minutes float64  `json:"minutes"`
	ColdKm  *float64 `json:"coldKm,omitempty"`
	HotKm   *float64 `json:"hotKm,omitempty"`
}

// tripRange: GET /api/trip-chain/range?reserve=20&target=80. How far one charge goes at highway speed between two battery levels,
// from how this car has actually driven (its usual efficiency at that speed, and in cold and hot weather when it has driven in
// them), so the longest leg can be set sensibly. The first window is the planner's own charge target and reserve.
func (s *Server) tripRange(w http.ResponseWriter, r *http.Request) {
	capacity := s.capacityKwh()
	base := s.whyBaseline()
	if s.fuelCar() || capacity == nil || base.Overall <= 0 {
		ok(w, map[string]any{"ready": false})
		return
	}
	num := func(k string, d float64) float64 {
		if v, err := strconv.ParseFloat(r.URL.Query().Get(k), 64); err == nil && v > 0 && v <= 100 {
			return v
		}
		return d
	}
	reserve, target := num("reserve", 20), num("target", 90)
	if target <= reserve {
		reserve, target = 20, 90
	}
	usual := base.ExpectedKmPerKwh(highwayKph, "")
	if usual <= 0 {
		usual = base.Overall
	}
	// the weather the car has really driven in: only offered when its own trips cover that temperature band
	weather := func(c float64) *float64 {
		g := insights.BandName(c)
		if _, ok := base.TempKmPKwh[g]; !ok || g == "" {
			return nil
		}
		v := base.ExpectedKmPerKwh(highwayKph, g)
		return &v
	}
	cold, hot := weather(-5), weather(35)
	var wins []rangeWindow
	seen := map[[2]float64]bool{}
	for _, p := range [][2]float64{{target, reserve}, {90, 20}, {80, 20}, {80, 10}} {
		if seen[p] || p[0] <= p[1] {
			continue
		}
		seen[p] = true
		kwh := (p[0] - p[1]) / 100 * *capacity
		win := rangeWindow{From: p[0], To: p[1], Km: kwh * usual}
		win.Minutes = win.Km / highwayKph * 60
		if cold != nil {
			v := kwh * *cold
			win.ColdKm = &v
		}
		if hot != nil {
			v := kwh * *hot
			win.HotKm = &v
		}
		wins = append(wins, win)
	}
	ok(w, map[string]any{"ready": true, "speedKph": highwayKph, "windows": wins, "capacityKwh": *capacity})
}
