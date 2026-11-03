package rebuild

import (
	"fmt"
	"sort"
)

// Group is one change a rebuild would make, so the person can accept or leave it: the stored trips or charges that go and the ones
// that take their place. A long trip that becomes three, or two that become one, is a single group; so is a trip that disappears
// (no new ones) or appears (no old ones).
type Group struct {
	Key  string `json:"key"`  // stable between the preview and the rebuild: the first old row's id, or the first new row's start
	Kind string `json:"kind"` // "trip" or "charge"
	Old  []int  `json:"old"`  // positions in OldTrips or OldCharges
	New  []int  `json:"new"`  // positions in NewTrips or NewCharges
}

type span struct{ a, b int64 }

// groupSpans joins the old and new rows of one kind that overlap in time (directly or through each other) into groups.
func groupSpans(kind string, olds, news []span, oldID func(int) int64) []Group {
	parent := make([]int, len(olds)+len(news))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i, o := range olds {
		for j, n := range news {
			if overlap(o.a, o.b, n.a, n.b) > 0 {
				parent[find(i)] = find(len(olds) + j)
			}
		}
	}
	byRoot := map[int]*Group{}
	var order []int
	add := func(node int, old bool, idx int) {
		r := find(node)
		g, ok := byRoot[r]
		if !ok {
			g = &Group{Kind: kind}
			byRoot[r] = g
			order = append(order, r)
		}
		if old {
			g.Old = append(g.Old, idx)
		} else {
			g.New = append(g.New, idx)
		}
	}
	for i := range olds {
		add(i, true, i)
	}
	for j := range news {
		add(len(olds)+j, false, j)
	}
	start := func(g *Group) int64 {
		s := int64(1 << 62)
		for _, i := range g.Old {
			s = min(s, olds[i].a)
		}
		for _, j := range g.New {
			s = min(s, news[j].a)
		}
		return s
	}
	out := make([]Group, 0, len(order))
	for _, r := range order {
		g := byRoot[r]
		if len(g.Old) > 0 {
			g.Key = fmt.Sprintf("%s:o%d", kind, oldID(g.Old[0]))
		} else {
			g.Key = fmt.Sprintf("%s:n%d", kind, news[g.New[0]].a)
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return start(&out[i]) < start(&out[j]) })
	return out
}

// Group lists every change in the plan, oldest first.
func (p *Plan) Group() []Group {
	var ot, nt, oc, nc []span
	for _, t := range p.OldTrips {
		ot = append(ot, span{t.StartTS, end(t)})
	}
	for _, t := range p.NewTrips {
		nt = append(nt, span{t.StartTS, end(t)})
	}
	for _, c := range p.OldCharges {
		oc = append(oc, span{c.StartTS, cEnd(c)})
	}
	for _, c := range p.NewCharges {
		nc = append(nc, span{c.StartTS, cEnd(c)})
	}
	gs := groupSpans("trip", ot, nt, func(i int) int64 { return p.OldTrips[i].ID })
	gs = append(gs, groupSpans("charge", oc, nc, func(i int) int64 { return p.OldCharges[i].ID })...)
	first := func(g Group) int64 {
		var s int64 = 1 << 62
		for _, i := range g.Old {
			if g.Kind == "trip" {
				s = min(s, p.OldTrips[i].StartTS)
			} else {
				s = min(s, p.OldCharges[i].StartTS)
			}
		}
		for _, j := range g.New {
			if g.Kind == "trip" {
				s = min(s, p.NewTrips[j].StartTS)
			} else {
				s = min(s, p.NewCharges[j].StartTS)
			}
		}
		return s
	}
	sort.SliceStable(gs, func(i, j int) bool { return first(gs[i]) < first(gs[j]) })
	return gs
}

// Only returns the plan with just the changes whose keys are given; the others stay as they are stored. The stretch and the
// blockers are kept, so a plan that cannot be applied still cannot.
func (p *Plan) Only(keys []string) *Plan {
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	q := *p
	q.OldTrips, q.NewTrips, q.OldCharges, q.NewCharges = nil, nil, nil, nil
	q.Changes = nil
	for _, g := range p.Group() {
		if !want[g.Key] {
			continue
		}
		for _, i := range g.Old {
			if g.Kind == "trip" {
				q.OldTrips = append(q.OldTrips, p.OldTrips[i])
			} else {
				q.OldCharges = append(q.OldCharges, p.OldCharges[i])
			}
		}
		for _, j := range g.New {
			if g.Kind == "trip" {
				q.NewTrips = append(q.NewTrips, p.NewTrips[j])
			} else {
				q.NewCharges = append(q.NewCharges, p.NewCharges[j])
			}
		}
	}
	return &q
}
