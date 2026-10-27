package charging

import ()

// Month is one calendar month of charging.
type Month struct {
	Month    string  `json:"month"` // 2026-09
	Sessions int     `json:"sessions"`
	Kwh      float64 `json:"kwh"`  // added to the battery
	Cost     float64 `json:"cost"` // of the sessions that have one
	Priced   int     `json:"priced"`
}

// ByMonth totals sessions by the month they started in the given zone, newest month first.
func ByMonth(ss []Session, monthOf func(int64) string) []Month {
	idx := map[string]*Month{}
	var order []string
	for _, s := range ss {
		if s.Open || s.AddedKwh == nil {
			continue
		}
		k := monthOf(s.StartTS)
		m := idx[k]
		if m == nil {
			m = &Month{Month: k}
			idx[k] = m
			order = append(order, k)
		}
		m.Sessions++
		m.Kwh += *s.AddedKwh
		if s.Cost != nil {
			m.Cost += *s.Cost
			m.Priced++
		}
	}
	out := make([]Month, 0, len(order))
	for _, k := range order {
		out = append(out, *idx[k])
	}
	for i := 0; i < len(out); i++ { // newest first; sessions arrive newest first already, but do not rely on it
		for j := i + 1; j < len(out); j++ {
			if out[j].Month > out[i].Month {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Learned is the overall efficiency of the slow-charging sessions where the charger's number was entered,
// weighted by energy, and how many sessions it rests on. Implausible entries are ignored.
func Learned(ss []Session) (eff float64, n int) {
	var added, wall float64
	for _, s := range ss {
		if !s.Measured || s.Efficiency == nil || s.PeakKw >= acKwLimit || *s.Efficiency < 0.6 || *s.Efficiency > 1 {
			continue
		}
		added += *s.AddedKwh
		wall += *s.WallKwh
		n++
	}
	if n == 0 || wall <= 0 {
		return 0, 0
	}
	return added / wall, n
}
