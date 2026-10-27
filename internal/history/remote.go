package history

import (
	"sort"
)

// Remote is a remote start for the feed.
type Remote struct {
	ID               int64
	Start, End       int64 // End 0 while it runs
	StartSoC, EndSoC *float64
	UsedKwh          *float64
	TempC            *float64
	Restarts         int
}

// WithRemote puts remote starts into a feed (newest first): a parked stretch that holds one is split around it, so the feed
// reads parked, remote start, parked. A remote start outside every parked stretch (it ended when the car was driven off) is
// added as it is.
func WithRemote(events []Event, rs []Remote, now int64) []Event {
	if len(rs) == 0 {
		return events
	}
	out := make([]Event, 0, len(events)+2*len(rs))
	used := map[int64]bool{}
	for _, e := range events {
		if e.Kind != "parked" {
			out = append(out, e)
			continue
		}
		pEnd := now
		if e.End != nil {
			pEnd = *e.End
		}
		cur := e.Start
		var pieces []Event
		for _, r := range rs {
			rEnd := r.End
			if rEnd == 0 {
				rEnd = now
			}
			if used[r.ID] || r.Start < e.Start-60 || r.Start >= pEnd {
				continue
			}
			used[r.ID] = true
			if r.Start-cur >= 60 {
				p := e
				p.Start, p.End = cur, ptr(r.Start)
				p.EndSoC = r.StartSoC
				pieces = append(pieces, p)
			}
			pieces = append(pieces, remoteEvent(r, e))
			cur = max(cur, rEnd)
			e.Cont = true // what follows is the same stop, starting with the battery the remote start left
			if r.EndSoC != nil {
				e.StartSoC = r.EndSoC
			}
		}
		if len(pieces) == 0 {
			out = append(out, e)
			continue
		}
		if pEnd-cur >= 60 || e.End == nil {
			p := e
			p.Start = cur
			pieces = append(pieces, p)
		}
		for i := len(pieces) - 1; i >= 0; i-- { // the feed is newest first
			out = append(out, pieces[i])
		}
	}
	for _, r := range rs {
		if !used[r.ID] {
			out = append(out, remoteEvent(r, Event{}))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start > out[j].Start })
	return out
}

func remoteEvent(r Remote, at Event) Event {
	e := Event{Kind: "remote", ID: r.ID, Start: r.Start, StartSoC: r.StartSoC, EndSoC: r.EndSoC, EnergyKwh: r.UsedKwh, TempC: r.TempC, Restarts: r.Restarts, Place: at.Place, Lat: at.Lat, Lon: at.Lon}
	if r.End > 0 {
		e.End = ptr(r.End)
	}
	return e
}

func ptr(v int64) *int64 { return &v }
