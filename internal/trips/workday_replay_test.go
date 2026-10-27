package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// A made-up work day: out from Home, a stop away, back Home, and a second outing in the evening.
const workDayFixture = `
0 OFF 1000 @45.5,-84.7
60 ON 1000 80 @45.5,-84.7
70 ON 1010 80 @45.6,-84.9
80 OFF 1015 @45.6,-84.9
100 ON 1015 80 @45.6,-84.9
110 ON 1025 80 @45.5,-84.7
115 OFF 1030 @45.5,-84.7
200 OFF 1030 @45.5,-84.7
201 ON 1030 70 @45.5,-84.7
210 ON 1036 70 @45.55,-84.8
215 OFF 1040 @45.55,-84.8`

func workDayReplay(t *testing.T) *replayer {
	r := newReplayer(t)
	if err := r.s.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45.5, Lon: -84.7, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	r.d.Work = func() WorkDay { return WorkDay{On: true, Days: 0b0111110, StaySec: 45 * 60} }
	r.feed(workDayFixture)
	return r
}

func cats(ts []*store.Trip) []string { // oldest first
	var out []string
	for i := len(ts) - 1; i >= 0; i-- {
		out = append(out, ts[i].Category)
	}
	return out
}

func TestAWorkDayIsOneWorkOutingAndAnEveningOutingIsPersonal(t *testing.T) {
	r := workDayReplay(t)
	got := cats(r.trips())
	if len(got) != 3 || got[0] != Work || got[1] != Work || got[2] != Personal {
		t.Fatalf("categories, oldest first: %v", got)
	}
	if src := r.trips()[0].CategorySource; src != "workday" {
		t.Fatalf("the source is named: %q", src)
	}
}

func TestReclassifyingFollowsTheOutingsAndLeavesHandLabelsAlone(t *testing.T) {
	r := workDayReplay(t)
	ts := r.trips()                                                                 // newest first: evening, afternoon, morning
	if err := r.s.SetTripCategory(ts[2].ID, "personal", "school run"); err != nil { // the owner relabels the first trip by hand
		t.Fatal(err)
	}
	if err := r.d.Reclassify(); err != nil {
		t.Fatal(err)
	}
	got := r.trips()
	if got[2].Category != Personal || got[2].CategorySource != "manual" {
		t.Fatalf("the hand label stays: %+v", got[2])
	}
	if got[1].Category != Personal { // the leg that carries on from it is the same outing
		t.Fatalf("the next leg follows the outing's label: %q", got[1].Category)
	}
	r.d.Work = nil
	if err := r.d.Reclassify(); err != nil {
		t.Fatal(err)
	}
	if got := r.trips(); got[0].Category != Personal {
		t.Fatalf("with the rule off the old rules apply: %q", got[0].Category)
	}
	_ = time.Second
}

func TestAnOutingSetByHandCoversTheTripsToComeUntilTheCarIsHome(t *testing.T) {
	r := newReplayer(t)
	if err := r.s.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45.5, Lon: -84.7, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	r.feed(`
0 OFF 1000 @45.5,-84.7
60 ON 1000 80 @45.5,-84.7
70 ON 1010 80 @45.6,-84.9
80 OFF 1015 @45.6,-84.9`)
	n, err := r.d.SetOuting(Work, r.t0.Add(85*time.Minute).Unix())
	if err != nil || n != 1 {
		t.Fatalf("the trip so far is relabeled: %d %v", n, err)
	}
	r.feed(`
100 ON 1015 80 @45.6,-84.9
110 ON 1025 80 @45.5,-84.7
115 OFF 1030 @45.5,-84.7
200 OFF 1030 @45.5,-84.7
201 ON 1030 70 @45.5,-84.7
210 ON 1036 70 @45.55,-84.8
215 OFF 1040 @45.55,-84.8`)
	got := cats(r.trips())
	if len(got) != 3 || got[0] != Work || got[1] != Work || got[2] != Personal {
		t.Fatalf("work until home, then the default: %v", got)
	}
	if r.trips()[1].CategorySource != "outing" {
		t.Fatalf("the later leg says where its label came from: %q", r.trips()[1].CategorySource)
	}
	if cat, _ := r.d.Outing(r.t0.Add(300 * time.Minute).Unix()); cat != "" {
		t.Fatalf("getting home ended the setting: %q", cat)
	}
}
