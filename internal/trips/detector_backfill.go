package trips

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// maxBackfillGap is the longest silence before a charge that is still treated as "the charge began in between": a parked car is
// polled every ten minutes, so a charge that starts right after a poll is first seen up to ten minutes late.
const maxBackfillGap = 60 * 60

// backfillStart moves the start of a new session back to the last reading before it when the battery had clearly risen since. A
// charge that begins between two polls is first seen part-way through, so without this its start level, energy and time are all
// too late and the session looks shorter than it was. The time is estimated from how long the charge power takes to add what the
// battery gained, and never earlier than that last reading.
func (d *Detector) backfillStart(c *store.Charge, st *ford.State) {
	now := st.TS.Unix()
	gap := now - d.lastReading
	if d.lastReading == 0 || d.lastCharging || d.lastSoC == nil || st.SoCPct == nil || gap <= 150 || gap > maxBackfillGap {
		return
	}
	if *st.SoCPct-*d.lastSoC < 2 {
		return // nothing came in that the first reading does not already show
	}
	start := d.lastReading
	c.StartKwh = d.lastKwh // from the same reading as the level; unknown rather than mixed with the later one
	if d.lastKwh != nil && st.EnergyKwh != nil {
		gained := *st.EnergyKwh - *d.lastKwh
		if gained < 1 {
			return
		}
		if st.ChargeKw != nil && *st.ChargeKw > 1 {
			if s := now - int64(gained / *st.ChargeKw * 3600); s > start {
				start = s
			}
		}
	}
	if start > now {
		start = now
	}
	d.S.LogDecision("charge_backfilled", fmt.Sprintf("a charge was first seen %d min after the last reading with the battery up from %.0f%% to %.0f%%; its start was moved back to about %s", gap/60, *d.lastSoC, *st.SoCPct, time.Unix(start, 0).In(d.loc()).Format("15:04")))
	c.StartTS, c.StartSoC = start, d.lastSoC
}

// noteChargerKwh keeps what the charger says it delivered, from the car's own count, when a session ends. Ford's count is only
// believed when it fits the energy that went into the battery (a charger delivers a little more than the pack gains), because a
// car can keep an old figure from an earlier session. A figure the owner typed is never replaced.
func (d *Detector) noteChargerKwh(c *store.Charge, st *ford.State) {
	// Only a fast charger: the count of an on-board (AC) charger may be the battery side, which would read as no loss at all and
	// spoil the measured slow-charge efficiency. A figure that is not the one this method kept earlier was typed, so it stays.
	if !strings.HasPrefix(c.Station, "DC") || c.StartKwh == nil || c.EndKwh == nil {
		return
	}
	if kept, auto := d.autoKwh[c.ID]; c.ChargerKwh != nil && (!auto || *c.ChargerKwh != kept) {
		return
	}
	gained := *c.EndKwh - *c.StartKwh
	v := ford.Number(st.Extra, "xevBatteryChargerEnergyOutput.value", "xevBatteryChargerEnergyOutput")
	if gained < 3 || v == nil || *v < gained*0.95 || *v > gained*1.35 {
		return
	}
	c.ChargerKwh = v
	if d.autoKwh == nil {
		d.autoKwh = map[int64]float64{}
	}
	d.autoKwh[c.ID] = *v // a paused session that resumes gets the final figure, not the one from the pause
	_ = d.S.SetChargerKwh(c.ID, v)
}
