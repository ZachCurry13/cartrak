package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/batteryhealth"
)

// batteryHealth is how the car's own battery-size estimate has moved over the last two years, in plain words.
func (s *Server) batteryHealth(w http.ResponseWriter, r *http.Request) {
	days, err := s.S.DailySeries("capacityKwh", time.Now().AddDate(-2, 0, 0).Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, batteryhealth.Build(days))
}
