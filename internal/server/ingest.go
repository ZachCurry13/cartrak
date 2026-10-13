package server

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	maxPointsPerCall = 1000
	callsPerMinute   = 300
	oldestAccepted   = 14 * 24 * time.Hour // queued offline points may be a few days old
)

// num reads a JSON number or a quoted number, because automation apps often build
// the body by hand and put quotes around everything.
type num float64

func (n *num) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("not a number")
	}
	*n = num(f)
	return nil
}

func ptrf(n *num) *float64 {
	if n == nil {
		return nil
	}
	f := float64(*n)
	return &f
}

// rateWindow limits each device to callsPerMinute calls a minute.
type rateWindow struct {
	mu   sync.Mutex
	seen map[int64][]time.Time
}

func (r *rateWindow) allow(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = map[int64][]time.Time{}
	}
	cut := time.Now().Add(-time.Minute)
	keep := r.seen[id][:0]
	for _, t := range r.seen[id] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	r.seen[id] = keep
	if len(keep) >= callsPerMinute {
		return false
	}
	r.seen[id] = append(keep, time.Now())
	return true
}

// deviceAuth accepts a device token ("Authorization: Bearer ...") instead of a login.
// Wrong tokens count against the same per-address lockout as wrong passwords.
func (s *Server) deviceAuth(next func(http.ResponseWriter, *http.Request, *store.Device)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := s.clientIP(r)
		if s.lim.blocked(ip) {
			fail(w, 429, "too many wrong tokens; try again in a minute")
			return
		}
		tok, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		var d *store.Device
		if found && tok != "" {
			d, _ = s.S.DeviceByHash(store.TokenHash(strings.TrimSpace(tok)))
		}
		if d == nil {
			s.lim.fail(ip)
			fail(w, 401, "unknown or removed device token")
			return
		}
		if !s.devRate.allow(d.ID) {
			fail(w, 429, "too many requests; send fewer, larger batches")
			return
		}
		next(w, r, d)
	}
}

// when turns a phone timestamp (seconds or milliseconds; 0 means now) into unix seconds.
func when(ts int64) (int64, bool) {
	now := time.Now()
	if ts == 0 {
		return now.Unix(), true
	}
	if ts > 100_000_000_000 {
		ts /= 1000
	}
	t := time.Unix(ts, 0)
	return ts, t.After(now.Add(-oldestAccepted)) && t.Before(now.Add(10*time.Minute))
}

type fixReq struct {
	TS    int64 `json:"ts"`
	Lat   *num  `json:"lat"`
	Lon   *num  `json:"lon"`
	Speed *num  `json:"speed"` // metres per second, as phones report it
	Acc   *num  `json:"acc"`   // metres
}

// point validates one fix. It returns false for one that cannot be stored.
func (f fixReq) point() (store.Point, bool) {
	ts, good := when(f.TS)
	if !good || f.Lat == nil || f.Lon == nil {
		return store.Point{}, false
	}
	lat, lon := float64(*f.Lat), float64(*f.Lon)
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 || (lat == 0 && lon == 0) {
		return store.Point{}, false
	}
	p := store.Point{TS: ts, Lat: lat, Lon: lon, AccM: ptrf(f.Acc)}
	if f.Speed != nil && *f.Speed >= 0 {
		kph := float64(*f.Speed) * 3.6
		p.SpeedKph = &kph
	}
	return p, true
}

type eventReq struct {
	fixReq
	Type string `json:"type"`
}

func (s *Server) ingestEvent(w http.ResponseWriter, r *http.Request, d *store.Device) {
	var in eventReq
	if err := decode(r, &in); err != nil {
		fail(w, 400, "the body must be JSON like {\"type\":\"drive_start\"}")
		return
	}
	switch in.Type {
	case "drive_start", "drive_end", "ping":
	default:
		fail(w, 400, "type must be drive_start, drive_end or ping")
		return
	}
	ts, good := when(in.TS)
	if !good {
		fail(w, 400, "ts is not a believable time (use unix seconds or milliseconds)")
		return
	}
	ev := store.Event{Device: d.ID, TS: ts, Type: in.Type}
	if p, ok := in.point(); ok {
		ev.Lat, ev.Lon = &p.Lat, &p.Lon
		if err := s.S.SavePoints(d.ID, []store.Point{p}); err != nil {
			fail(w, 500, "could not save the location")
			return
		}
	}
	if err := s.S.AddEvent(ev); err != nil {
		fail(w, 500, "could not save the event")
		return
	}
	_ = s.S.TouchDevice(d.ID, in.Type)
	if s.P != nil && in.Type != "ping" {
		s.P.PhoneDriving(in.Type == "drive_start")
	}
	ok(w, map[string]any{"ok": true})
}

func (s *Server) ingestPoints(w http.ResponseWriter, r *http.Request, d *store.Device) {
	var in struct {
		Points []fixReq `json:"points"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, 400, "the body must be JSON like {\"points\":[{\"lat\":1,\"lon\":2}]}")
		return
	}
	if len(in.Points) > maxPointsPerCall {
		fail(w, 400, "at most "+strconv.Itoa(maxPointsPerCall)+" points per call")
		return
	}
	keep := make([]store.Point, 0, len(in.Points))
	for _, f := range in.Points {
		if p, good := f.point(); good {
			keep = append(keep, p)
		}
	}
	if len(keep) > 0 {
		if err := s.S.SavePoints(d.ID, keep); err != nil {
			fail(w, 500, "could not save the points")
			return
		}
	}
	_ = s.S.TouchDevice(d.ID, "")
	ok(w, map[string]int{"saved": len(keep), "skipped": len(in.Points) - len(keep)})
}
