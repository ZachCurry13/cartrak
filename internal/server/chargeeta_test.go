package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheChargingEstimateLearnsTheTaperFromEarlierSessions(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	get := func(target int) (code int, out struct {
		Minutes        float64
		Learned        bool
		Sessions       int
		Level          string
		Idle           bool
		Unavailable    bool
		Preconditioned bool
	}) {
		c, b := do(t, admin, "GET", fmt.Sprintf("%s/api/charges/eta?target=%d", ts.URL, target), "", nil)
		_ = json.Unmarshal([]byte(b), &out)
		return c, out
	}
	if _, o := get(80); !o.Idle {
		t.Fatalf("nothing is charging: %+v", o)
	}
	kwAt := func(soc float64) float64 { // 9 kW, falling to 3 kW between 90 and 100 percent
		if soc < 90 {
			return 9
		}
		return 9 - (soc-90)*0.6
	}
	base := time.Now().Add(-10 * 24 * time.Hour).Truncate(time.Minute)
	for s := 0; s < 3; s++ { // three finished sessions
		start := base.Add(time.Duration(s) * 24 * time.Hour)
		n := 0
		for soc := 20.0; soc <= 100; soc += 2 {
			st := &ford.State{TS: start.Add(time.Duration(n) * 10 * time.Minute), SoCPct: fp(soc), ChargeKw: fp(kwAt(soc)), Plug: "Connected", ChargeState: "Charging", CapacityKwh: fp(88), Raw: []byte(`{}`)}
			if _, err := srv.S.SaveState("t", st); err != nil {
				t.Fatal(err)
			}
			n++
		}
		end := start.Add(time.Duration(n) * 10 * time.Minute).Unix()
		c := &store.Charge{StartTS: start.Unix(), EndTS: &end, StartSoC: fp(20), EndSoC: fp(100), Open: false}
		if err := srv.S.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		c.EndTS, c.Open = &end, false // CreateCharge starts an open session; finish it
		if err := srv.S.UpdateCharge(c); err != nil {
			t.Fatal(err)
		}
	}
	now := &ford.State{TS: time.Now().Add(-time.Minute), SoCPct: fp(60), ChargeKw: fp(9), Plug: "Connected", ChargeState: "Charging", CapacityKwh: fp(88), Raw: []byte(`{}`)}
	if _, err := srv.S.SaveState("t", now); err != nil {
		t.Fatal(err)
	}
	code, o := get(100)
	if code != 200 || !o.Learned || o.Sessions != 3 || o.Level != "l2" {
		t.Fatalf("learned from three level 2 sessions: %d %+v", code, o)
	}
	flat := 88 * 0.40 / 9 * 60
	if o.Minutes < flat*1.05 {
		t.Fatalf("the taper near full makes 60 to 100 percent longer than %.0f min, got %.0f", flat, o.Minutes)
	}
	if _, o80 := get(80); !o80.Learned || o80.Minutes > flat*0.55 {
		t.Fatalf("to 80 percent there is no taper: %+v", o80)
	}
}

func TestMatchingAnotherMileageAppSetsTheOdometerCorrection(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour).Unix()
	mk := func(a, b float64) *store.Trip {
		tr := &store.Trip{StartTS: start.Unix(), Category: "personal", CategorySource: "default", StartOdo: fp(a)}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.EndOdo, tr.Open, tr.DistanceKm = &end, fp(b), false, b-a
		srv.S.UpdateTrip(tr)
		return tr
	}
	long, short := mk(1000, 1103.9), mk(2000, 2001)
	post := func(id int64, km float64) (int, string) {
		return do(t, admin, "POST", fmt.Sprintf("%s/api/trips/%d/calibrate", ts.URL, id), fmt.Sprintf(`{"km":%g}`, km), xh)
	}
	if code, b := post(long.ID, 101.2); code != 200 {
		t.Fatalf("%d %s", code, b)
	}
	if got := srv.Cfg.OdometerFactor(); got < 0.973 || got > 0.975 {
		t.Fatalf("101.2 km measured against 103.9 km on the odometer is about 0.974: %.4f", got)
	}
	if code, _ := post(short.ID, 1); code != 400 {
		t.Fatalf("a one kilometer trip is too short to compare: %d", code)
	}
	if code, _ := post(long.ID, 200); code != 400 {
		t.Fatalf("a figure 90%% off is refused: %d", code)
	}
}
