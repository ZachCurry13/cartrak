package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// Every reading carries around 50 KB of raw fields. The few numbers the Vehicle health page
// charts are therefore kept in a small table of their own, so a chart never has to read through
// the big rows (it was slow enough to stall the whole app on a long-running database).

const kvBackfill = "trend_backfill_id"

// saveTrend records the charted values of one reading, under the same id as its sample.
func (s *Store) saveTrend(id int64, st *ford.State) error {
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO trends(id,ts,battery12v,soc,hv_volts,capacity_kwh,oil_pct,fuel_pct) VALUES(?,?,?,?,?,?,?,?)`,
		id, st.TS.Unix(), st.Battery12V, st.SoCPct, st.HVVolts, st.CapacityKwh, st.OilLifePct, st.FuelPct)
	return err
}

// BackfillTrends copies the charted values of readings saved before the trends table existed.
// It works in small batches and pauses between them, so the app stays responsive, and it
// remembers where it got to. It returns when everything is copied or ctx ends.
func (s *Store) BackfillTrends(ctx context.Context, batch int, pause time.Duration) error {
	last, _ := strconv.ParseInt(s.GetKV(kvBackfill), 10, 64)
	for ctx.Err() == nil {
		rows, err := s.DB.Query(`SELECT id,ts,battery12v_v,soc_pct,hv_volts,capacity_kwh,oil_pct,fuel_pct FROM samples WHERE id>? ORDER BY id LIMIT ?`, last, batch)
		if err != nil {
			return err
		}
		n := 0
		type row struct {
			id, ts                          int64
			v12, soc, hv, capKwh, oil, fuel sql.NullFloat64
		}
		var got []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.ts, &r.v12, &r.soc, &r.hv, &r.capKwh, &r.oil, &r.fuel); err != nil {
				rows.Close()
				return err
			}
			got = append(got, r)
		}
		rows.Close()
		for _, r := range got {
			if _, err := s.DB.Exec(`INSERT OR IGNORE INTO trends(id,ts,battery12v,soc,hv_volts,capacity_kwh,oil_pct,fuel_pct) VALUES(?,?,?,?,?,?,?,?)`,
				r.id, r.ts, r.v12, r.soc, r.hv, r.capKwh, r.oil, r.fuel); err != nil {
				return err
			}
			last, n = r.id, n+1
		}
		if n > 0 {
			_ = s.SetKV(kvBackfill, strconv.FormatInt(last, 10))
		}
		if n < batch {
			return nil
		}
		select {
		case <-ctx.Done():
		case <-time.After(pause):
		}
	}
	return ctx.Err()
}
