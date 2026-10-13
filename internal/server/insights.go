package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/insights"
)

// efficiencyInsights is how efficiently the car has been driven over the last year: by month, by outside
// temperature, and the best and worst trips.
func (s *Server) efficiencyInsights(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -366).Unix()
	ts, err := s.S.EnergyTrips(from)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	temps, err := s.S.TripTemps(from)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, insights.Build(ts, s.capacityKwh(), temps, s.defaultLoc()))
}
