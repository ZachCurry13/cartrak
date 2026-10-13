package store

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// Every reading carries around 50 KB of raw fields. The numbers that charts and timelines need are
// therefore kept in a small table of their own, so browsing history never has to read through the
// big rows (that was slow enough to stall the whole app on a long-running database).

const kvBackfill = "trend_backfill2_id" // "2": the table gained its timeline columns

const trendCols = `id,ts,battery12v,soc,hv_volts,capacity_kwh,oil_pct,fuel_pct,range_km,odometer_km,speed_kph,outside_c,driving,charging`

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// saveTrend records the charted values of one reading, under the same id as its sample.
func (s *Store) saveTrend(id int64, st *ford.State) error {
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO trends(`+trendCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, st.TS.Unix(), st.Battery12V, st.SoCPct, st.HVVolts, st.CapacityKwh, st.OilLifePct, st.FuelPct,
		st.RangeKm, st.OdometerKm, st.SpeedKph, st.OutsideC, b2i(st.Driving()), b2i(st.Charging()))
	return err
}

type backfillRow struct {
	id, ts                                                  int64
	v12, soc, hv, capKwh, oil, fuel, rng, odo, spd, out, kw sql.NullFloat64
	ign, chargeState                                        sql.NullString
}

// BackfillTrends copies the charted values of readings saved before the trends table was complete.
// It works in small batches and pauses between them, so the app stays responsive, and it
// remembers where it got to. It returns when everything is copied or ctx ends.
func (s *Store) BackfillTrends(ctx context.Context, batch int, pause time.Duration) error {
	last, _ := strconv.ParseInt(s.GetKV(kvBackfill), 10, 64)
	for ctx.Err() == nil {
		got, err := s.backfillBatch(last, batch)
		if err != nil {
			return err
		}
		for _, r := range got {
			if err := s.writeBackfilled(r); err != nil {
				return err
			}
			last = r.id
		}
		if len(got) > 0 {
			_ = s.SetKV(kvBackfill, strconv.FormatInt(last, 10))
		}
		if len(got) < batch {
			return nil
		}
		select {
		case <-ctx.Done():
		case <-time.After(pause):
		}
	}
	return ctx.Err()
}

func (s *Store) backfillBatch(after int64, batch int) ([]backfillRow, error) {
	rows, err := s.DB.Query(`SELECT id,ts,battery12v_v,soc_pct,hv_volts,capacity_kwh,oil_pct,fuel_pct,range_km,odometer_km,speed_kph,outside_c,
		charge_kw,ignition,charge_state FROM samples WHERE id>? ORDER BY id LIMIT ?`, after, batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var got []backfillRow
	for rows.Next() {
		var r backfillRow
		if err := rows.Scan(&r.id, &r.ts, &r.v12, &r.soc, &r.hv, &r.capKwh, &r.oil, &r.fuel, &r.rng, &r.odo, &r.spd, &r.out, &r.kw, &r.ign, &r.chargeState); err != nil {
			return nil, err
		}
		got = append(got, r)
	}
	return got, rows.Err()
}

func (s *Store) writeBackfilled(r backfillRow) error {
	flags := &ford.State{Ignition: r.ign.String, ChargeState: r.chargeState.String}
	if r.spd.Valid {
		flags.SpeedKph = &r.spd.Float64
	}
	if r.kw.Valid {
		flags.ChargeKw = &r.kw.Float64
	}
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO trends(`+trendCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.id, r.ts, r.v12, r.soc, r.hv, r.capKwh, r.oil, r.fuel, r.rng, r.odo, r.spd, r.out, b2i(flags.Driving()), b2i(flags.Charging()))
	return err
}
