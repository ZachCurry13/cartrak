package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/tirewatch"
)

// tireTrends is each tire against the placard pressure the car reports, and whether one is slowly going down.
func (s *Server) tireTrends(w http.ResponseWriter, r *http.Request) {
	rows, err := s.S.TirePoints(time.Now().AddDate(0, 0, -30).Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	pts := make([]tirewatch.Point, 0, len(rows))
	for _, x := range rows {
		pts = append(pts, tirewatch.Point{TS: x.TS, P: x.P, OutsideC: x.OutsideC})
	}
	var front, rear *float64
	if x, _ := s.S.LatestSample(); x != nil {
		f := map[string]any{}
		_ = json.Unmarshal(x.Extra, &f)
		front, rear = ford.Number(f, "wheelPlacardFront"), ford.Number(f, "wheelPlacardRear")
	}
	ok(w, tirewatch.Build(pts, front, rear))
}
