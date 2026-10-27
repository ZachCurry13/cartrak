package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/remotestart"
)

// BackfillRemoteStarts goes through the readings saved before remote starts were recorded, a batch at a time with a pause
// between, remembering where it got to. upTo is the newest reading it should look at (those after it were seen live).
func (s *Store) BackfillRemoteStarts(ctx context.Context, upTo int64, batch int, pause time.Duration) error {
	if err := s.ensureRemote(); err != nil {
		return err
	}
	last, _ := strconv.ParseInt(s.GetKV(kvRemoteBackfill), 10, 64)
	if last >= upTo {
		return nil
	}
	var open *remotestart.Session
	var before *remotestart.Reading
	for ctx.Err() == nil {
		rows, err := s.DB.Query(`SELECT id,ts,soc_pct,energy_kwh,outside_c,speed_kph,extra FROM samples WHERE id>? AND id<=? ORDER BY id LIMIT ?`, last, upTo, batch)
		if err != nil {
			return err
		}
		// read the whole batch first: the database has one connection, and writing while rows are open would wait forever
		type row struct {
			id, ts              int64
			soc, kwh, temp, spd sql.NullFloat64
			raw                 sql.NullString
		}
		var got []row
		for rows.Next() {
			var x row
			if err := rows.Scan(&x.id, &x.ts, &x.soc, &x.kwh, &x.temp, &x.spd, &x.raw); err != nil {
				rows.Close()
				return err
			}
			got = append(got, x)
		}
		rows.Close()
		for _, x := range got {
			last = x.id
			extra := map[string]any{}
			_ = json.Unmarshal([]byte(x.raw.String), &extra)
			r := readingOf(x.ts, extra, nullPtr(x.soc), nullPtr(x.kwh), nullPtr(x.temp), x.spd.Valid && x.spd.Float64 > 1)
			var ended *remotestart.Session
			open, ended = remotestart.Step(open, r, before)
			if ended != nil {
				ended.ID = 0
				s.saveRemote(ended)
			}
			before = &r
		}
		n := len(got)
		_ = s.SetKV(kvRemoteBackfill, strconv.FormatInt(last, 10))
		if n < batch || last >= upTo {
			return nil // a start still running at the end is left to the live readings
		}
		select {
		case <-ctx.Done():
		case <-time.After(pause):
		}
	}
	return ctx.Err()
}

// LatestSampleID is the id of the newest reading, 0 when there is none.
func (s *Store) LatestSampleID() int64 {
	var id sql.NullInt64
	_ = s.DB.QueryRow(`SELECT MAX(id) FROM samples`).Scan(&id)
	return id.Int64
}
