package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/remotestart"
)

// Remote starts are found as readings arrive (the car counts down while one runs) and kept in a small table, so the Day view
// and History never read the big raw readings. Readings saved before this existed are gone through once, gently.

const remoteDDL = `CREATE TABLE IF NOT EXISTS remote_starts (
  id INTEGER PRIMARY KEY, start_ts INTEGER NOT NULL, end_ts INTEGER, start_soc REAL, end_soc REAL, start_kwh REAL, end_kwh REAL,
  temp_c REAL, restarts INTEGER NOT NULL DEFAULT 0, drove INTEGER NOT NULL DEFAULT 0, last_ts INTEGER, last_left REAL);
CREATE INDEX IF NOT EXISTS remote_starts_ts ON remote_starts(start_ts);`

const kvRemoteBackfill = "remote_backfill_id"

type remoteState struct {
	mu     sync.Mutex
	before *remotestart.Reading
	table  sync.Once
	err    error
}

var remoteStates sync.Map // *Store -> *remoteState

func (s *Store) remote() *remoteState {
	v, _ := remoteStates.LoadOrStore(s, &remoteState{})
	return v.(*remoteState)
}

// ensureRemote makes the table once per database.
func (s *Store) ensureRemote() error {
	rs := s.remote()
	rs.table.Do(func() { _, rs.err = s.DB.Exec(remoteDDL) })
	return rs.err
}

func readingOf(ts int64, extra map[string]any, soc, kwh, temp *float64, driving bool) remotestart.Reading {
	r := remotestart.Reading{TS: ts, SoC: soc, Kwh: kwh, TempC: temp, Driving: driving}
	if v := ford.Number(extra, "remoteStartCountdownTimer"); v != nil {
		r.Left = *v
	}
	if v := ford.Number(extra, "remoteStartRunDurationSetting"); v != nil {
		r.RunFor = *v
	}
	return r
}

// noteRemote applies a new reading to the remote start running now, if any.
func (s *Store) noteRemote(st *ford.State) {
	rs := s.remote()
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r := readingOf(st.TS.Unix(), st.Extra, st.SoCPct, st.EnergyKwh, st.OutsideC, st.Driving())
	s.applyRemote(r, rs.before)
	rs.before = &r
}

func (s *Store) applyRemote(r remotestart.Reading, before *remotestart.Reading) {
	if s.ensureRemote() != nil {
		return
	}
	open := s.openRemote()
	if open == nil && r.Left <= 0 {
		return // the usual case: nothing running, nothing to write
	}
	running, ended := remotestart.Step(open, r, before)
	if ended != nil {
		s.saveRemote(ended)
	}
	if running != nil {
		s.saveRemote(running)
	}
}

func (s *Store) openRemote() *remotestart.Session {
	var x remotestart.Session
	var lastTS sql.NullInt64
	var lastLeft sql.NullFloat64
	err := s.DB.QueryRow(`SELECT id,start_ts,start_soc,end_soc,start_kwh,end_kwh,temp_c,restarts,last_ts,last_left FROM remote_starts WHERE end_ts IS NULL ORDER BY id DESC LIMIT 1`).
		Scan(&x.ID, &x.Start, &x.StartSoC, &x.EndSoC, &x.StartKwh, &x.EndKwh, &x.TempC, &x.Restarts, &lastTS, &lastLeft)
	if err != nil {
		return nil
	}
	x.LastTS, x.LastLeft = lastTS.Int64, lastLeft.Float64
	return &x
}

func (s *Store) saveRemote(x *remotestart.Session) {
	var end any
	if x.End > 0 {
		end = x.End
	}
	if x.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO remote_starts(start_ts,end_ts,start_soc,end_soc,start_kwh,end_kwh,temp_c,restarts,drove,last_ts,last_left) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			x.Start, end, x.StartSoC, x.EndSoC, x.StartKwh, x.EndKwh, x.TempC, x.Restarts, b2i(x.Drove), x.LastTS, x.LastLeft)
		if err == nil {
			x.ID, _ = r.LastInsertId()
		}
		return
	}
	_, _ = s.DB.Exec(`UPDATE remote_starts SET end_ts=?,end_soc=?,end_kwh=?,temp_c=?,restarts=?,drove=?,last_ts=?,last_left=? WHERE id=?`,
		end, x.EndSoC, x.EndKwh, x.TempC, x.Restarts, b2i(x.Drove), x.LastTS, x.LastLeft, x.ID)
}

// RemoteStartsBetween is every remote start that overlaps from..to, oldest first.
func (s *Store) RemoteStartsBetween(from, to int64) ([]remotestart.Session, error) {
	if err := s.ensureRemote(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,start_ts,COALESCE(end_ts,0),start_soc,end_soc,start_kwh,end_kwh,temp_c,restarts,drove FROM remote_starts
		WHERE start_ts<=? AND COALESCE(end_ts,?)>=? ORDER BY start_ts`, to, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []remotestart.Session
	for rows.Next() {
		var x remotestart.Session
		var drove int
		if err := rows.Scan(&x.ID, &x.Start, &x.End, &x.StartSoC, &x.EndSoC, &x.StartKwh, &x.EndKwh, &x.TempC, &x.Restarts, &drove); err != nil {
			return nil, err
		}
		x.Drove = drove == 1
		out = append(out, x)
	}
	return out, rows.Err()
}

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

func nullPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}
