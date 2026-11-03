package rebuild

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func trip(id, a, b int64) *store.Trip { return &store.Trip{ID: id, StartTS: a, EndTS: &b} }

func TestAChangeJoinsTheOldAndNewRowsThatOverlapAndNothingElse(t *testing.T) {
	p := &Plan{
		// a road trip stored as one long trip that the replay splits in two, a short trip that is stored but should not exist,
		// and a trip the replay finds that was never stored
		OldTrips: []*store.Trip{trip(7, 0, 1000), trip(9, 5000, 5100)},
		NewTrips: []*store.Trip{trip(0, 0, 400), trip(0, 600, 1000), trip(0, 8000, 8200)},
	}
	gs := p.Group()
	if len(gs) != 3 {
		t.Fatalf("one split, one removal, one addition: %+v", gs)
	}
	if g := gs[0]; g.Key != "trip:o7" || len(g.Old) != 1 || len(g.New) != 2 {
		t.Errorf("the split: %+v", g)
	}
	if g := gs[1]; g.Key != "trip:o9" || len(g.Old) != 1 || len(g.New) != 0 {
		t.Errorf("the removal: %+v", g)
	}
	if g := gs[2]; g.Key != "trip:n8000" || len(g.Old) != 0 || len(g.New) != 1 {
		t.Errorf("the addition: %+v", g)
	}
}

func TestOnlyKeepsTheTickedChangesAndLeavesTheRestAsStored(t *testing.T) {
	p := &Plan{
		From: 0, To: 9000, Blockers: []string{},
		OldTrips: []*store.Trip{trip(7, 0, 1000), trip(9, 5000, 5100)},
		NewTrips: []*store.Trip{trip(0, 0, 400), trip(0, 600, 1000), trip(0, 8000, 8200)},
	}
	q := p.Only([]string{"trip:o7"})
	if len(q.OldTrips) != 1 || q.OldTrips[0].ID != 7 || len(q.NewTrips) != 2 {
		t.Fatalf("only the split is applied: old %d new %d", len(q.OldTrips), len(q.NewTrips))
	}
	if q := p.Only([]string{}); len(q.OldTrips)+len(q.NewTrips) != 0 {
		t.Fatal("nothing ticked, nothing applied")
	}
	if len(p.OldTrips) != 2 || len(p.NewTrips) != 3 {
		t.Fatal("the plan itself is not changed")
	}
	if q := p.Only([]string{"trip:o9", "trip:n8000"}); len(q.OldTrips) != 1 || len(q.NewTrips) != 1 {
		t.Fatalf("a removal and an addition: old %d new %d", len(q.OldTrips), len(q.NewTrips))
	}
}

func TestChargesAreGroupedLikeTrips(t *testing.T) {
	e := int64(100)
	p := &Plan{OldCharges: []*store.Charge{{ID: 3, StartTS: 0, EndTS: &e}}, NewCharges: []*store.Charge{{StartTS: 10, EndTS: &e}}}
	gs := p.Group()
	if len(gs) != 1 || gs[0].Kind != "charge" || gs[0].Key != "charge:o3" {
		t.Fatalf("%+v", gs)
	}
}
