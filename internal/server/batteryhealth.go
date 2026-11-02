package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/batteryhealth"
)

// batteryHealth is how the car's own battery-size estimate has moved over the last two years, in plain words. It uses the middle
// reading of each day, which is steadier than the average.
func (s *Server) batteryHealth(w http.ResponseWriter, r *http.Request) {
	days, err := s.S.DailyMedian("capacityKwh", time.Now().AddDate(-2, 0, 0).Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	rep := batteryhealth.Build(days)
	if s.Cfg != nil {
		rep.Against(s.Cfg.BatteryNewKwh())
	} else {
		rep.Against(0)
	}
	ok(w, rep)
}
