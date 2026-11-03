package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

// calibrateOdometer sets the odometer correction so that this trip's distance becomes what another app measured for it. The factor
// is the other app's distance over the raw odometer distance, so it is the same whatever correction was in force before.
// POST /api/trips/{id}/calibrate {km}. Admin only.
func (s *Server) calibrateOdometer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct{ Km float64 }
	t, err := s.S.GetTrip(id)
	if decode(r, &in) != nil || err != nil || t == nil || t.Open || t.StartOdo == nil || t.EndOdo == nil {
		fail(w, 404, "no finished trip with an odometer reading")
		return
	}
	raw := *t.EndOdo - *t.StartOdo
	if raw < 3 {
		fail(w, 400, "that trip is too short to compare: pick one of at least 2 miles (3 km)")
		return
	}
	f := in.Km / raw
	if !(f >= 0.8 && f <= 1.2) {
		fail(w, 400, "that is more than 20% away from the odometer: check the number, or use the correction under Settings")
		return
	}
	if err := s.Cfg.Set(settings.OdometerFactorKey, fmt.Sprintf("%.4f", f)); err != nil {
		fail(w, 400, err.Error())
		return
	}
	ok(w, map[string]any{"factor": f})
}
