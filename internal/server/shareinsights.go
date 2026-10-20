package server

import "github.com/ZachCurry13/cartrak/internal/store"

// shareInsights is the efficiency detail a link can add: how fast the car went, and for a single trip the temperature and what made it
// use more or less energy than usual. Nothing in it says where the car was or what the trips were for.
func (s *Server) shareInsights(ts []*store.Trip) map[string]any {
	out := map[string]any{}
	var top, km float64
	var moving int64
	for _, t := range ts {
		if t.MaxSpeedKph > top {
			top = t.MaxSpeedKph
		}
		km += t.DistanceKm
		if t.EndTS != nil {
			moving += *t.EndTS - t.StartTS
		}
	}
	out["topKph"] = top
	if moving > 0 {
		out["avgKph"] = km / (float64(moving) / 3600) // over the whole time, stops included, so it is honest for a day
	}
	if len(ts) == 1 {
		if rep, err := s.whyFor(ts[0]); err == nil && rep.Ready {
			out["vsUsualPct"] = rep.VsUsualPct
			out["tempC"] = rep.Facts.TempC
			out["avgMovingKph"] = rep.Facts.AvgKph
			out["stops"] = rep.Facts.Stops
			tags := []map[string]string{}
			for _, t := range rep.Tags {
				tags = append(tags, map[string]string{"label": t.Label, "level": t.Level})
			}
			out["tags"] = tags
		}
	}
	return out
}
