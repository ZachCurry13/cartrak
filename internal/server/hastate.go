package server

import (
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// haState is a small, read-only snapshot of the car for Home Assistant or any other smart-home tool, read with
// a device token (the same kind phone tracking uses). It never includes the VIN or any location.
func (s *Server) haState(w http.ResponseWriter, r *http.Request, _ *store.Device) {
	x, err := s.S.LatestSample()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	if x == nil {
		ok(w, map[string]any{"ready": false})
		return
	}
	plug := strings.ToLower(x.Plug)
	var plugged *bool
	if plug != "" {
		v := strings.Contains(plug, "connect") && !strings.Contains(plug, "disconnect") && !strings.Contains(plug, "not")
		plugged = &v
	}
	charging := ford.ChargeStateCharging(x.ChargeState) || (x.ChargeKw != nil && *x.ChargeKw > 0.2)
	if x.ChargeKw != nil && *x.ChargeKw > 0.2 {
		charging = true
	}
	ok(w, map[string]any{
		"ready": true, "ts": x.TS, "ageSeconds": time.Now().Unix() - x.TS,
		"batteryPct": x.SoCPct, "rangeKm": x.RangeKm, "energyKwh": x.EnergyKwh, "capacityKwh": x.CapacityKwh,
		"plugged": plugged, "charging": charging, "chargeKw": x.ChargeKw,
		"locked": x.Locked, "ignition": x.Ignition, "speedKph": x.SpeedKph,
		"odometerKm": x.OdometerKm, "battery12vVolts": x.Battery12V, "outsideC": x.OutsideC,
	})
}
