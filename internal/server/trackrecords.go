package server

import (
	"encoding/json"
	"io"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// trackRecords writes the trips and charging sessions that overlap the stretch, one JSON object per line.
func (s *Server) trackRecords(file func(string, func(io.Writer)), from, to int64, private bool) {
	file("trips.jsonl", func(f io.Writer) {
		ts, _ := s.S.ListTrips(store.TripFilter{From: from - 6*3600, To: to + 1, All: true})
		for i := len(ts) - 1; i >= 0; i-- {
			t := *ts[i]
			if t.EndTS != nil && *t.EndTS < from {
				continue
			}
			if !private {
				t.StartLat, t.StartLon, t.EndLat, t.EndLon = nil, nil, nil, nil
			}
			b, _ := json.Marshal(t)
			_, _ = f.Write(append(b, '\n'))
		}
	})
	file("charges.jsonl", func(f io.Writer) {
		cs, _ := s.S.ListCharges(300)
		for i := len(cs) - 1; i >= 0; i-- {
			c := *cs[i]
			if c.StartTS > to || (c.EndTS != nil && *c.EndTS < from) {
				continue
			}
			if !private {
				c.Lat, c.Lon = nil, nil
			}
			b, _ := json.Marshal(c)
			_, _ = f.Write(append(b, '\n'))
		}
	})
}
