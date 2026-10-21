package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/remotestart"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// remoteStarts is how many remote starts CarTrak has seen since the car was last driven. The car allows only a couple in
// a row, so it helps to know. GET /api/remote-starts
func (s *Server) remoteStarts(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	from := now - 3*86400 // far enough back to cover a weekend away; no drive in that time means "this stretch"
	var lastDrive int64
	if list, err := s.S.ListTrips(store.TripFilter{Limit: 1}); err == nil && len(list) > 0 {
		t := list[0]
		end := t.StartTS
		if t.EndTS != nil {
			end = *t.EndTS
		}
		if end > from {
			from = end
		}
		lastDrive = end
	}
	rows, err := s.S.SamplesBetween(from, now)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var pts []remotestart.Point
	for _, x := range rows {
		f := map[string]any{}
		if json.Unmarshal(x.Extra, &f) != nil {
			continue
		}
		left := 0.0
		if v := ford.Number(f, "remoteStartCountdownTimer"); v != nil {
			left = *v
		}
		pts = append(pts, remotestart.Point{TS: x.TS, Left: left})
	}
	ok(w, map[string]any{"sinceDrive": remotestart.Count(pts), "since": from, "lastDrive": lastDrive})
}

// remoteFor is the remote starts that overlap from..to, ready for the feed.
func (s *Server) remoteFor(from, to int64) []history.Remote {
	list, err := s.S.RemoteStartsBetween(from, to)
	if err != nil {
		return nil
	}
	out := make([]history.Remote, 0, len(list))
	for _, x := range list {
		out = append(out, history.Remote{ID: x.ID, Start: x.Start, End: x.End, StartSoC: x.StartSoC, EndSoC: x.EndSoC, UsedKwh: x.UsedKwh(), TempC: x.TempC, Restarts: x.Restarts})
	}
	return out
}
