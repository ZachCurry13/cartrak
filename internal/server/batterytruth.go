package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/batteryhealth"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type truthNow struct {
	DisplayedSoC *float64 `json:"displayedSoc"` // what the car shows
	ActualSoC    *float64 `json:"actualSoc"`    // the battery's own level
	EnergyKwh    *float64 `json:"energyKwh"`
	UsableKwh    *float64 `json:"usableKwh"`  // energy left divided by the true level
	FordKwh      *float64 `json:"fordKwh"`    // the capacity Ford reports
	MaxRangeKm   *float64 `json:"maxRangeKm"` // range at 100 percent
}

// usable works the usable battery size out from one reading: the energy left divided by the true charge level. Near
// the top and the bottom the level is too coarse to divide by, so those readings give nothing.
func usable(f map[string]any) (float64, bool) {
	soc, kwh := ford.Number(f, "xevBatteryActualStateOfCharge"), ford.Number(f, "xevBatteryEnergyRemaining")
	if soc == nil || kwh == nil || *soc < 20 || *soc > 95 || *kwh <= 0 {
		return 0, false
	}
	return *kwh / (*soc / 100), true
}

var (
	truthMu    sync.Mutex
	truthAt    time.Time
	truthCache []store.Day
)

// truthSeries is the usable size, one reading a day for the last half year, kept for six hours (it reads the raw
// field list of about a hundred readings).
func (s *Server) truthSeries() []store.Day {
	truthMu.Lock()
	defer truthMu.Unlock()
	if time.Since(truthAt) < 6*time.Hour && truthCache != nil {
		return truthCache
	}
	var out []store.Day
	seen := map[int64]bool{}
	now := time.Now()
	for d := 180; d >= 0; d-- {
		noon := time.Date(now.Year(), now.Month(), now.Day()-d, 12, 0, 0, 0, s.defaultLoc())
		x, err := s.S.SampleAt(noon.Unix())
		if err != nil || x == nil || seen[x.TS] || noon.Sub(time.Unix(x.TS, 0)) > 36*time.Hour {
			continue
		}
		seen[x.TS] = true
		f := map[string]any{}
		if json.Unmarshal(x.Extra, &f) != nil {
			continue
		}
		if u, ok := usable(f); ok {
			day := noon.Truncate(24*time.Hour).Unix() / 86400 * 86400
			out = append(out, store.Day{TS: day, Avg: u, Min: u, Max: u, N: 1})
		}
	}
	truthCache, truthAt = out, time.Now()
	if truthCache == nil {
		truthCache = []store.Day{}
	}
	return truthCache
}

// batteryTruth compares what the car shows with the battery's own figures, and tracks the usable size over time.
func (s *Server) batteryTruth(w http.ResponseWriter, r *http.Request) {
	var now truthNow
	if x, _ := s.S.LatestSample(); x != nil {
		f := map[string]any{}
		_ = json.Unmarshal(x.Extra, &f)
		now.DisplayedSoC = ford.Number(f, "xevBatteryStateOfCharge")
		now.ActualSoC = ford.Number(f, "xevBatteryActualStateOfCharge")
		now.EnergyKwh = ford.Number(f, "xevBatteryEnergyRemaining")
		now.FordKwh = ford.Number(f, "xevBatteryCapacity")
		now.MaxRangeKm = ford.Number(f, "xevBatteryMaximumRange")
		if u, ok := usable(f); ok {
			now.UsableKwh = &u
		}
	}
	ok(w, map[string]any{"now": now, "usable": batteryhealth.Build(s.truthSeries())})
}
