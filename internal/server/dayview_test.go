package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestADayCombinesItsTripsRoutesAndStopsInOrder(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Date(2026, 10, 18, 0, 0, 0, 0, loc)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	fp := func(v float64) *float64 { return &v }
	mkTrip := func(start, end time.Time, a, b [2]float64, km float64) {
		for i, p := range [][2]float64{a, {(a[0] + b[0]) / 2, (a[1] + b[1]) / 2}, b} {
			st := &ford.State{TS: start.Add(time.Duration(i) * (end.Sub(start) / 2)), Lat: fp(p[0]), Lon: fp(p[1]), OdometerKm: fp(1000), Raw: []byte(`{}`)}
			if _, err := srv.S.SaveState("t", st); err != nil {
				t.Fatal(err)
			}
		}
		tr := &store.Trip{StartTS: start.Unix(), StartLat: fp(a[0]), StartLon: fp(a[1]), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		e := end.Unix()
		tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &e, fp(b[0]), fp(b[1]), km, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	mkTrip(at(7, 0), at(8, 30), [2]float64{44.0, -85.0}, [2]float64{44.5, -85.5}, 80)
	// A fast charge at the end of the first drive, then a second drive.
	c := &store.Charge{StartTS: at(8, 35).Unix(), Lat: fp(44.5), Lon: fp(-85.5), StartSoC: fp(20), StartKwh: fp(18), PeakKw: 120}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	end := at(9, 5).Unix()
	c.EndTS, c.EndSoC, c.EndKwh, c.Open = &end, fp(80), fp(70), false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}
	mkTrip(at(9, 10), at(10, 40), [2]float64{44.5, -85.5}, [2]float64{45.0, -86.0}, 85)
	code, b := do(t, admin, "GET", ts.URL+"/api/day?date=2026-10-18", "", nil)
	var out struct {
		Date   string
		Events []struct{ Kind string }
		Routes []struct {
			Kind   string
			Points [][]float64
		}
		Totals  struct{ Km float64 }
		Summary struct {
			Trips, Sessions, FastSessions int
			TopKph                        float64
			ChargeSec                     int64
			Unpriced                      int
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	kinds := ""
	for _, e := range out.Events {
		kinds += e.Kind + ","
	}
	if out.Date != "2026-10-18" || kinds != "driving,charging,driving,parked," {
		t.Fatalf("the day reads forwards: drive, charge, drive: %s", kinds)
	}
	if len(out.Routes) != 2 || out.Routes[0].Kind != "car" || len(out.Routes[0].Points) < 2 || out.Totals.Km != 165 {
		t.Fatalf("two routes and 165 km: %+v", out)
	}
	if out.Summary.Trips != 2 || out.Summary.Sessions != 1 || out.Summary.FastSessions != 1 || out.Summary.ChargeSec != 30*60 {
		t.Fatalf("the summary counts two drives and one fast charge of half an hour: %+v", out.Summary)
	}
	if code, _ := do(t, admin, "GET", fmt.Sprintf("%s/api/day?date=nope", ts.URL), "", nil); code != 400 {
		t.Fatalf("a bad date: %d", code)
	}
}
