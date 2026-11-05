package server

import (
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	restIdleMin = 30 // minutes with nothing going on before a 12 V reading counts as resting
	restDays    = 14 // how far back it looks
	restKeep    = 10 * time.Minute
)

type restingDay struct {
	TS  int64   `json:"ts"`
	Avg float64 `json:"avg"`
	N   int     `json:"n"`
}

// restingVolts is the 12 V battery's voltage when the car has been sitting: not driving, not charging and nothing happening for at
// least half an hour. The reading taken while the car is awake is the charging system's, around 14 V, and says little about the battery;
// the resting figure is what shows how it holds up.
type restingVolts struct {
	Avg     float64      `json:"avg"`
	Min     float64      `json:"min"`
	Max     float64      `json:"max"`
	N       int          `json:"n"`
	Days    []restingDay `json:"days"`
	Latest  float64      `json:"latest"`
	LatestT int64        `json:"latestTs"`
	IdleMin int          `json:"idleMin"`
	Span    int          `json:"spanDays"`
}

type restEntry struct {
	at time.Time
	v  *restingVolts
}

// restKept holds the last answer for each car (a server can hold several), for a few minutes.
var (
	restMu   sync.Mutex
	restKept = map[int64]restEntry{}
)

// awake says the car was doing something at that reading: ignition on, moving, or charging.
func awake(x *store.Sample) bool {
	ig := strings.ToLower(x.Ignition)
	if strings.Contains(ig, "run") || strings.Contains(ig, "start") || strings.Contains(ig, "acc") || ig == "on" {
		return true
	}
	return (x.SpeedKph != nil && *x.SpeedKph > 1) || (x.ChargeKw != nil && *x.ChargeKw > 0.2)
}

// restingVoltage works the resting figure out from the readings of the last two weeks; nil when there are none yet.
func (s *Server) restingVoltage() *restingVolts {
	restMu.Lock()
	defer restMu.Unlock()
	if e, ok := restKept[s.CarID]; ok && time.Since(e.at) < restKeep {
		return e.v
	}
	now := time.Now()
	rows, err := s.S.SamplesBetween(now.AddDate(0, 0, -restDays).Unix(), now.Unix())
	if err != nil || len(rows) == 0 {
		return nil
	}
	loc := s.defaultLoc()
	byDay := map[string]*restingDay{}
	var order []string
	out := &restingVolts{IdleMin: restIdleMin, Span: restDays, Min: 99}
	var sum float64
	lastBusy := int64(0)
	for _, x := range rows {
		if awake(x) {
			lastBusy = x.TS
			continue
		}
		if x.Battery12V == nil || lastBusy == 0 || x.TS-lastBusy < restIdleMin*60 || *x.Battery12V < 5 {
			continue
		}
		v := *x.Battery12V
		sum += v
		out.N++
		out.Min, out.Max = min(out.Min, v), max(out.Max, v)
		out.Latest, out.LatestT = v, x.TS
		k := time.Unix(x.TS, 0).In(loc).Format("2006-01-02")
		d := byDay[k]
		if d == nil {
			d = &restingDay{TS: x.TS}
			byDay[k] = d
			order = append(order, k)
		}
		d.Avg = (d.Avg*float64(d.N) + v) / float64(d.N+1)
		d.N++
	}
	if out.N == 0 {
		restKept[s.CarID] = restEntry{at: now}
		return nil
	}
	out.Avg = sum / float64(out.N)
	for _, k := range order {
		out.Days = append(out.Days, *byDay[k])
	}
	restKept[s.CarID] = restEntry{at: now, v: out}
	return out
}
