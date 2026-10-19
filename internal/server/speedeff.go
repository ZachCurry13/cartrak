package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/speedeff"
)

// speedEfficiency is how much energy the car uses at each speed, from real drives. GET /api/speed-efficiency?days=90
func (s *Server) speedEfficiency(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 7 || days > 365 {
		days = 90
	}
	rows, err := s.S.DrivingReadings(time.Now().AddDate(0, 0, -days).Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	rs := make([]speedeff.Reading, 0, len(rows))
	for _, x := range rows {
		rs = append(rs, speedeff.Reading{TS: int64(x[0]), Odo: x[1], Kwh: x[2], Speed: x[3]})
	}
	bands := speedeff.Build(rs)
	if bands == nil {
		bands = []speedeff.Out{}
	}
	ok(w, map[string]any{"days": days, "bands": bands})
}
