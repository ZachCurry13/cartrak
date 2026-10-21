package history

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }

func TestParkedTimeFillsTheGapsAndTheFeedIsNewestFirst(t *testing.T) {
	home := []*store.NamedPlace{{Name: "Home", Lat: 40, Lon: -80, RadiusM: 200}}
	trips := []*store.Trip{{ID: 1, StartTS: 1000, EndTS: i64(2000), DistanceKm: 20, StartSoC: f64(80), EndSoC: f64(70), EndLat: f64(40), EndLon: f64(-80)}}
	cs := []Charge{{ID: 9, Start: 5000, End: i64(8000), StartSoC: f64(70), EndSoC: f64(90), AddedKwh: f64(15), Cost: f64(2)}}
	ev := Build(trips, cs, home, 0, 100000, 9000)
	if len(ev) != 4 {
		t.Fatalf("drive, parked, charge, parked: %+v", ev)
	}
	if ev[0].Kind != "parked" || ev[0].End != nil || ev[0].Start != 8000 {
		t.Fatalf("the newest is the stretch still going: %+v", ev[0])
	}
	if ev[1].Kind != "charging" || ev[2].Kind != "parked" || ev[3].Kind != "driving" {
		t.Fatalf("order: %v %v %v", ev[1].Kind, ev[2].Kind, ev[3].Kind)
	}
	if ev[2].Place != "Home" || ev[2].StartSoC == nil || *ev[2].StartSoC != 70 || ev[2].EndSoC == nil || *ev[2].EndSoC != 70 {
		t.Fatalf("parked at home, battery 70 to 70: %+v", ev[2])
	}
	tot := Sum(ev, 0, 100000, 9000)
	if tot.Km != 20 || tot.Kwh != 15 || tot.Cost != 2 || tot.DrivingSec != 1000 || tot.ChargingSec != 3000 || tot.ParkedSec != 3000+1000 {
		t.Fatalf("totals: %+v", tot)
	}
}

func TestShortGapsAreNotShownAndAnOpenDriveHasNoParkedAfterIt(t *testing.T) {
	trips := []*store.Trip{{ID: 1, StartTS: 1000, EndTS: i64(2000)}, {ID: 2, StartTS: 2100, Open: true}}
	ev := Build(trips, nil, nil, 0, 100000, 9000)
	if len(ev) != 2 {
		t.Fatalf("two drives and no parked: %+v", ev)
	}
}

func TestARemoteStartSplitsTheParkedTimeAroundIt(t *testing.T) {
	end := int64(10000)
	ev := []Event{{Kind: "driving", ID: 2, Start: 10000, End: &[]int64{11000}[0]}, {Kind: "parked", Start: 1000, End: &end, Place: "Home"}}
	used := 1.2
	out := WithRemote(ev, []Remote{{ID: 7, Start: 9000, End: 9600, UsedKwh: &used}}, 20000)
	if len(out) != 4 || out[0].Kind != "driving" || out[1].Kind != "parked" || out[2].Kind != "remote" || out[3].Kind != "parked" {
		t.Fatalf("drive, parked, remote start, parked: %+v", out)
	}
	if out[2].Place != "Home" || *out[2].EnergyKwh != 1.2 || *out[3].End != 9000 || out[1].Start != 9600 || !out[1].Cont || out[3].Cont || out[1].StartSoC != nil {
		t.Fatalf("split: %+v", out)
	}
	if got := WithRemote(ev, nil, 20000); len(got) != 2 {
		t.Fatal("nothing to add")
	}
}
