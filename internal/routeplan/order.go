package routeplan

import "math"

// Best is the order to visit places in so the total driving time is least: it starts at place 0 and visits every other place once.
// With roundTrip it also drives back to place 0 at the end; without, it ends wherever is quickest. times[i][j] is the minutes
// from i to j. The answer is exact (a search over every subset), which is quick for the ten or so places a day has.
func Best(times [][]float64, roundTrip bool) []int {
	n := len(times)
	if n <= 2 {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	m := n - 1 // the places other than the start, as bits 0..m-1 for places 1..n-1
	full := 1<<m - 1
	cost := make([][]float64, 1<<m)
	from := make([][]int8, 1<<m)
	for s := range cost {
		cost[s], from[s] = make([]float64, m), make([]int8, m)
		for j := range cost[s] {
			cost[s][j], from[s][j] = math.Inf(1), -1
		}
	}
	for j := 0; j < m; j++ {
		cost[1<<j][j] = times[0][j+1]
	}
	for s := 1; s <= full; s++ {
		for j := 0; j < m; j++ {
			if s&(1<<j) == 0 || math.IsInf(cost[s][j], 1) {
				continue
			}
			for k := 0; k < m; k++ {
				if s&(1<<k) != 0 {
					continue
				}
				ns := s | 1<<k
				if c := cost[s][j] + times[j+1][k+1]; c < cost[ns][k] {
					cost[ns][k], from[ns][k] = c, int8(j)
				}
			}
		}
	}
	end, best := 0, math.Inf(1)
	for j := 0; j < m; j++ {
		c := cost[full][j]
		if roundTrip {
			c += times[j+1][0]
		}
		if c < best {
			end, best = j, c
		}
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
	return order
}

// TotalMinutes is the driving time of visiting the places in this order, and back to the first when roundTrip.
func TotalMinutes(times [][]float64, order []int, roundTrip bool) float64 {
	var t float64
	for i := 1; i < len(order); i++ {
		t += times[order[i-1]][order[i]]
	}
	if roundTrip && len(order) > 1 {
		t += times[order[len(order)-1]][order[0]]
	}
	return t
}
