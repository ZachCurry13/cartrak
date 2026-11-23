package server

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeplan"
)

// maxStayMin is the longest time at one stop the planner takes (a whole working day).
const maxStayMin = 600.0

// visits is what each stop asks of the day: the time spent there (its own, else the planner's "time at each stop") and its set time,
// in minutes after leaving. The start asks for nothing.
func visits(stops []stopPoint, depart int64, stayMin float64) []routeplan.Visit {
	v := make([]routeplan.Visit, len(stops))
	for i, p := range stops {
		v[i] = routeplan.Visit{StayMin: stayMin, AtMin: routeplan.NoTime}
		if p.StayMin != nil {
			v[i].StayMin = math.Max(0, math.Min(*p.StayMin, maxStayMin))
		}
		if p.AtTS > 0 && p.AtTS > depart-12*3600 {
			v[i].AtMin = math.Max(0, float64(p.AtTS-depart)/60)
		}
		if i == 0 {
			v[i] = routeplan.Visit{AtMin: routeplan.NoTime}
		}
	}
	return v
}

// timedOrder puts the destinations in the order that ends the day soonest while every set time is kept, and the last stop last when
// asked. It returns the stops in that order, where each came from, the minutes of driving it saves, and whether every set time is made.
func (s *Server) timedOrder(ctx context.Context, roads bool, stops []stopPoint, v []routeplan.Visit, roundTrip, endLast bool) ([]stopPoint, []int, float64, bool) {
	times, _ := s.drivingTimes(ctx, roads, stops)
	order, onTime := routeplan.BestTimed(times, v, roundTrip, endLast)
	identity := make([]int, len(stops))
	for i := range identity {
		identity[i] = i
	}
	if order == nil {
		return stops, identity, 0, false
	}
	saved := routeplan.TotalMinutes(times, identity, roundTrip) - routeplan.TotalMinutes(times, order, roundTrip)
	same := true
	for i, k := range order {
		same = same && k == i
	}
	if same {
		return stops, identity, 0, onTime
	}
	out := make([]stopPoint, len(order))
	for i, k := range order {
		out[i] = stops[k]
	}
	return out, order, math.Max(0, saved), onTime
}

// stopTimes is, for the destination of each leg, the wait for its set time, how late it is reached and the time spent there; and
// the warnings for the set times that are missed.
type stopTime struct {
	AtTS    int64   `json:"atTs,omitempty"`
	WaitMin float64 `json:"waitMin"`
	LateMin float64 `json:"lateMin"`
	StayMin float64 `json:"stayMin"`
}

// arriveAt applies a stop's set time and stay to an arrival at t (unix seconds): the moment it is left, and what happened there.
func arriveAt(v routeplan.Visit, p stopPoint, depart, t int64) (int64, stopTime) {
	a := routeplan.Arrival{ArriveMin: float64(t-depart) / 60}
	st := stopTime{StayMin: v.StayMin}
	if v.AtMin >= 0 {
		st.AtTS = p.AtTS
		if a.ArriveMin < v.AtMin {
			st.WaitMin = v.AtMin - a.ArriveMin
		} else {
			st.LateMin = a.ArriveMin - v.AtMin
		}
	}
	return t + int64((st.WaitMin+st.StayMin)*60), st
}

func lateWarning(name string, st stopTime, loc *time.Location) string {
	return fmt.Sprintf("%s is reached about %d min after its set time (%s). Leave earlier, or move a stop to after it.", name,
		int(math.Round(st.LateMin)), time.Unix(st.AtTS, 0).In(loc).Format("3:04 pm"))
}
