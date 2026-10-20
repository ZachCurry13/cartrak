package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/cardata"
	"github.com/ZachCurry13/cartrak/internal/fieldwatch"
	"github.com/ZachCurry13/cartrak/internal/ford"
)

type fieldOut struct {
	fieldwatch.Stat
	UsedFor string `json:"usedFor"` // what CarTrak uses it for beyond showing it, or ""
	Shown   bool   `json:"shown"`   // it is on the Car data page
}

// fieldWatch is everything the car reports and what each thing has been doing lately: how often it changes, when it last
// did, and whether CarTrak does anything with it. GET /api/fields?hours=48
func (s *Server) fieldWatch(w http.ResponseWriter, r *http.Request) {
	hours := intQuery(r, "hours", 48, 1, 168)
	now := time.Now().Unix()
	rows, err := s.S.SamplesBetween(now-int64(hours)*3600, now)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	step := 1
	if len(rows) > 400 {
		step = len(rows)/400 + 1 // thinned: a thousand readings of the same thing say no more than four hundred
	}
	var in []fieldwatch.Row
	var latest map[string]any
	for i, x := range rows {
		last := i == len(rows)-1
		if i%step != 0 && !last {
			continue
		}
		f := ford.Flatten(x.Extra)
		in = append(in, fieldwatch.Row{TS: x.TS, Flat: f})
		if last {
			latest = f
		}
	}
	shown := map[string]bool{}
	if latest != nil {
		rep := cardata.Build(latest, time.Unix(now, 0), ford.Powertrain(s.vehicleInfo().Powertrain))
		for _, t := range rep.Tabs {
			for _, g := range t.Groups {
				for _, row := range g.Rows {
					name := row.Raw
					if i := strings.IndexAny(name, ".["); i > 0 {
						name = name[:i]
					}
					shown[name] = true
				}
			}
		}
	}
	stats := fieldwatch.Analyze(in)
	out := make([]fieldOut, 0, len(stats))
	for _, st := range stats {
		out = append(out, fieldOut{Stat: st, UsedFor: fieldwatch.Features[st.Metric], Shown: shown[st.Metric]})
	}
	ok(w, map[string]any{"hours": hours, "readings": len(rows), "fields": out, "now": now})
}
