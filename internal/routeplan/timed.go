package routeplan

import "math"

// Visit is what a place asks of the day: how long is spent there and, for some (lunch at noon), when to be there. AtMin is minutes
// after leaving the start; a negative AtMin means no set time.
type Visit struct {
	StayMin float64
	AtMin   float64
}

// NoTime is a Visit's AtMin when the place has no set time.
const NoTime = -1.0

// OnTimeSlack is how many minutes after its set time a place may be reached and still count as on time.
const OnTimeSlack = 10.0

// Arrival is one place on the day: when it is reached, how long is waited for its set time, how late it is, and when it is left.
type Arrival struct {
	ArriveMin, WaitMin, LateMin, LeaveMin float64
}

// reach is arriving at a place at t: wait for its set time when early, then stay.
func reach(v Visit, t float64) Arrival {
	a := Arrival{ArriveMin: t}
	if v.AtMin >= 0 {
		if t < v.AtMin {
			a.WaitMin = v.AtMin - t
		} else {
			a.LateMin = t - v.AtMin
		}
	}
	a.LeaveMin = t + a.WaitMin + v.StayMin
	return a
}

// Schedule walks the places in this order from the start (left at minute 0): each one's arrival, wait, lateness and leaving time,
// with the way back to the start last when roundTrip.
func Schedule(times [][]float64, v []Visit, order []int, roundTrip bool) []Arrival {
	out := make([]Arrival, 0, len(order)+1)
	t := 0.0
	for i := 1; i < len(order); i++ {
		a := reach(v[order[i]], t+times[order[i-1]][order[i]])
		out = append(out, a)
		t = a.LeaveMin
	}
	if roundTrip && len(order) > 1 {
		out = append(out, Arrival{ArriveMin: t + times[order[len(order)-1]][order[0]], LeaveMin: t + times[order[len(order)-1]][order[0]]})
	}
	return out
}

// BestTimed is the order that ends the day soonest while every place with a set time is reached by then (within OnTimeSlack),
// waiting when early. It starts at place 0; with endLast (and not roundTrip) the last place given stays last. With no set times it is
// the order of least driving, as Best gives. ok is false when no order makes every set time: then the order given is the one with
// the least lateness in all.
func BestTimed(times [][]float64, v []Visit, roundTrip, endLast bool) (order []int, ok bool) {
	if o, done := bestTimed(times, v, roundTrip, endLast, false); done {
		return o, true
	}
	o, _ := bestTimed(times, v, roundTrip, endLast, true)
	return o, false
}

// bestTimed searches every subset (exact for the ten or so places of a day). Each state keeps the earliest moment it can be left; with
// late, lateness is allowed and counted first, so the order found is the one that is least late in all.
func bestTimed(times [][]float64, v []Visit, roundTrip, endLast, late bool) ([]int, bool) {
	n := len(times)
	if n <= 2 {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out, n < 2 || late || reach(v[1], times[0][1]).LateMin <= OnTimeSlack
	}
	m := n - 1
	last := -1 // the place that must be last, as a bit index
	if endLast && !roundTrip {
		last = m - 1
	}
	full := 1<<m - 1
	type cell struct{ lateSum, leave float64 }
	better := func(a, b cell) bool {
		if a.lateSum != b.lateSum {
			return a.lateSum < b.lateSum
		}
		return a.leave < b.leave
	}
	inf := cell{math.Inf(1), math.Inf(1)}
	cost := make([][]cell, 1<<m)
	from := make([][]int8, 1<<m)
	for s := range cost {
		cost[s], from[s] = make([]cell, m), make([]int8, m)
		for j := range cost[s] {
			cost[s][j], from[s][j] = inf, -1
		}
	}
	try := func(ns, k int, prev cell, t float64, j int8) {
		a := reach(v[k+1], t)
		if !late && a.LateMin > OnTimeSlack {
			return
		}
		c := cell{prev.lateSum, a.LeaveMin}
		if late {
			c.lateSum += a.LateMin
		}
		if better(c, cost[ns][k]) {
			cost[ns][k], from[ns][k] = c, j
		}
	}
	for k := 0; k < m; k++ {
		if k != last || m == 1 {
			try(1<<k, k, cell{}, times[0][k+1], -1)
		}
	}
	for s := 1; s <= full; s++ {
		for j := 0; j < m; j++ {
			c := cost[s][j]
			if s&(1<<j) == 0 || math.IsInf(c.leave, 1) || j == last && s != full {
				continue
			}
			for k := 0; k < m; k++ {
				ns := s | 1<<k
				if s&(1<<k) != 0 || (k == last && ns != full) {
					continue
				}
				try(ns, k, c, c.leave+times[j+1][k+1], int8(j))
			}
		}
	}
	end, best := -1, inf
	for j := 0; j < m; j++ {
		if last >= 0 && j != last {
			continue
		}
		c := cost[full][j]
		if roundTrip {
			c.leave += times[j+1][0]
		}
		if !math.IsInf(c.leave, 1) && better(c, best) {
			end, best = j, c
		}
	}
	if end < 0 {
		return nil, false
	}
	order := make([]int, 0, n)
	for s, j := full, end; j >= 0; {
		order = append(order, j+1)
		pj := int(from[s][j])
		s &^= 1 << j
		j = pj
	}
	order = append(order, 0)
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}
	return order, true
}
