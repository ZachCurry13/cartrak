package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// A remote start with the car plugged in runs the climate control from the charger, and the car reports it as charging: a few
// minutes at a few kW that add no energy to the battery. It is not a charge, so it is marked, and the pages show it as a remote start.
const (
	remoteSlack  = 5 * time.Minute // a session may start or end this far from the remote start it belongs to
	remoteMaxKwh = 0.5             // a session that added less than this is nothing but heating or cooling
)

// markRemote sets Remote on the sessions that are a remote start with the car plugged in.
func (s *Server) markRemote(ss []charging.Session) {
	if len(ss) == 0 {
		return
	}
	from, to := ss[0].StartTS, time.Now().Unix()
	for _, c := range ss {
		from = min(from, c.StartTS)
	}
	rs, err := s.S.RemoteStartsBetween(from-int64(remoteSlack.Seconds()), to+int64(remoteSlack.Seconds()))
	if err != nil || len(rs) == 0 {
		return
	}
	slack := int64(remoteSlack.Seconds())
	for i := range ss {
		c := &ss[i]
		if c.AddedKwh != nil && *c.AddedKwh >= remoteMaxKwh {
			continue
		}
		cEnd := to
		if c.EndTS != nil {
			cEnd = *c.EndTS
		}
		for _, r := range rs {
			rEnd := r.End
			if rEnd == 0 {
				rEnd = to
			}
			if c.StartTS <= rEnd+slack && cEnd >= r.Start-slack {
				c.Remote = true
				break
			}
		}
	}
}

// withoutRemote is the sessions that are real charges.
func withoutRemote(ss []charging.Session) []charging.Session {
	out := make([]charging.Session, 0, len(ss))
	for _, c := range ss {
		if !c.Remote {
			out = append(out, c)
		}
	}
	return out
}
