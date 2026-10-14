package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/cardata"
	"github.com/ZachCurry13/cartrak/internal/ford"
)

// sampleFor finds the reading the page asked for: the newest at or before ?at=<unix seconds>, or the
// latest one when no time is given.
func (s *Server) sampleFor(r *http.Request) (at int64, ts int64, f map[string]any, found bool) {
	at, _ = strconv.ParseInt(r.URL.Query().Get("at"), 10, 64)
	x, err := s.S.LatestSample()
	if at > 0 {
		x, err = s.S.SampleAt(at)
	}
	if err != nil || x == nil {
		return at, 0, nil, false
	}
	f = map[string]any{}
	_ = json.Unmarshal(x.Extra, &f)
	return at, x.TS, f, true
}

// carData is the Car data page: the car as it was at a moment (the latest by default), in plain words,
// with the times of the readings either side so the page can step back and forward.
func (s *Server) carData(w http.ResponseWriter, r *http.Request) {
	at, ts, f, found := s.sampleFor(r)
	first := s.S.FirstSampleTS()
	if !found {
		ok(w, map[string]any{"ready": false, "first": first, "requested": at})
		return
	}
	info := s.vehicleInfo()
	prev, next := s.S.NeighbourTS(ts)
	ok(w, map[string]any{
		"ready": true, "ts": ts, "requested": at, "latest": at == 0 || next == 0,
		"prev": prev, "next": next, "first": first, "model": modelName(f), "vehicle": info,
		"trimmed": len(f) == 0, // the full detail of old readings can be removed to save space (More, Backup)
		"report": cardata.Build(f, time.Unix(ts, 0), ford.Powertrain(info.Powertrain)),
	})
}

// carDataRaw is every value of one reading, for the All fields tab (loaded only when that tab opens).
func (s *Server) carDataRaw(w http.ResponseWriter, r *http.Request) {
	_, ts, f, found := s.sampleFor(r)
	if !found {
		ok(w, map[string]any{"ready": false})
		return
	}
	ok(w, map[string]any{"ready": true, "ts": ts, "fields": f})
}

// timeline gives the readings of one day (?day=2026-10-13, in the car's time zone) or any ?from= &to=
// range, thinned for a chart, from the small trends table.
func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	loc := time.UTC
	if s.Cfg != nil {
		loc = s.Cfg.Location()
	}
	q := r.URL.Query()
	from, _ := strconv.ParseInt(q.Get("from"), 10, 64)
	to, _ := strconv.ParseInt(q.Get("to"), 10, 64)
	if from == 0 || to == 0 {
		day := time.Now().In(loc)
		if d := q.Get("day"); d != "" {
			t, err := time.ParseInLocation("2006-01-02", d, loc)
			if err != nil {
				fail(w, 400, "the day must look like 2026-10-13")
				return
			}
			day = t
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
		from, to = start.Unix(), start.AddDate(0, 0, 1).Unix()-1
	}
	if to < from || to-from > 400*86400 {
		fail(w, 400, "choose a range of up to 400 days")
		return
	}
	pts, err := s.S.Timeline(from, to, 400)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"from": from, "to": to, "points": pts})
}
