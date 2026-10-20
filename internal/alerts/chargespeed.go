package alerts

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	slowShare   = 0.7  // a fast charge averaging less than this share of your usual is slow
	slowMinSess = 3    // earlier fast sessions needed to know what usual is
	slowMaxFrom = 60.0 // a session that started above this is expected to taper, so it is not judged
	slowMinMin  = 10   // minutes a session must last to be judged
	slowMaxEnd  = 85.0 // a session that ran above this spent its last part tapering, which lowers the average on any charger
	kvSlow      = "alert_slow_charge_id"
)

// fastAvg is the average power of a finished fast session, from the energy that went in over the time it took. ok is false
// when it is not a fast session or lacks the numbers.
func fastAvg(c *store.Charge) (kw float64, ok bool) {
	if c.Open || c.EndTS == nil || c.StartKwh == nil || c.EndKwh == nil || *c.EndKwh <= *c.StartKwh {
		return 0, false
	}
	fast := c.PeakKw >= 40 || len(c.Station) >= 2 && c.Station[:2] == "DC"
	mins := float64(*c.EndTS-c.StartTS) / 60
	if !fast || mins < slowMinMin {
		return 0, false
	}
	return (*c.EndKwh - *c.StartKwh) / (mins / 60), true
}

// judgeable is whether a fast session can be compared by its average power: it began low enough to be at full speed and ended
// before the taper near full. A charge to 100% always averages less than one to 80%, however good the charger is.
func judgeable(c *store.Charge) bool {
	return c.StartSoC != nil && *c.StartSoC <= slowMaxFrom && (c.EndSoC == nil || *c.EndSoC <= slowMaxEnd)
}

// chargeSpeed tells you when a fast charge that has just ended gave much less than your fast charges usually do.
func (e *Engine) chargeSpeed() {
	cs, err := e.S.ListCharges(40)
	if err != nil || len(cs) == 0 {
		return
	}
	cur := cs[0] // newest first
	kw, ok := fastAvg(cur)
	if !ok || !judgeable(cur) {
		return
	}
	if e.S.GetKV(kvSlow) == strconv.FormatInt(cur.ID, 10) {
		return
	}
	var prev []float64
	for _, c := range cs[1:] {
		if v, ok := fastAvg(c); ok && judgeable(c) {
			prev = append(prev, v)
		}
	}
	if len(prev) < slowMinSess {
		return
	}
	sort.Float64s(prev)
	usual := prev[len(prev)/2]
	if kw >= usual*slowShare {
		return
	}
	_ = e.S.SetKV(kvSlow, strconv.FormatInt(cur.ID, 10))
	e.send(push.Alert{Kind: push.ChargeSlow, Title: "Slow fast charge", Body: fmt.Sprintf("That session averaged %.0f kW. Your fast charges usually average about %.0f kW. Another charger or stall may be faster.", kw, usual), URL: "#/charging"})
}
