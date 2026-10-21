package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheTankIsLearnedFromAFillUpAndTripsGetGallonsAndCost(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-6 * time.Hour).Truncate(time.Second).Unix()
	level := func(at int64, pct float64) { srv.S.DB.Exec(`INSERT INTO trends(ts,fuel_pct) VALUES(?,?)`, at, pct) }
	// a fill-up from 30% to 90% that took 8.4 gallons: a 14 gallon tank
	level(t0, 30)
	level(t0+600, 90)
	pending := &fuel.Fill{TS: t0 + 600, StartPct: 30, EndPct: 90, Status: "pending", Source: "auto"}
	_ = srv.S.SaveRefuel(pending)

	_, body := do(t, admin, "GET", ts.URL+"/api/fuel", "", nil)
	if !strings.Contains(body, `"tankGal":null`) || !strings.Contains(body, `"pending":1`) {
		t.Fatalf("no tank size until a fill-up has its gallons: %.300s", body)
	}
	for _, bad := range []string{`{"gallons":0,"pricePerGal":3}`, `{"gallons":500,"pricePerGal":3}`, `{"gallons":8,"pricePerGal":-1}`} {
		if code, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/fuel/fills/%d", ts.URL, pending.ID), bad, xh); code != 400 {
			t.Fatalf("%s should be refused", bad)
		}
	}
	if code, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/fuel/fills/%d", ts.URL, pending.ID), `{"gallons":8.4,"pricePerGal":3.25}`, xh); code != 200 {
		t.Fatal("complete the fill-up")
	}
	var page struct {
		TankGal   float64
		TankFills int
		Pending   int
		Fills     []struct{ Cost float64 }
		Months    []struct{ Gallons, Cost float64 }
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/fuel", "", nil)
	if json.Unmarshal([]byte(body), &page) != nil || math.Abs(page.TankGal-14) > 0.01 || page.TankFills != 1 || page.Pending != 0 ||
		len(page.Fills) != 1 || math.Abs(page.Fills[0].Cost-27.3) > 0.01 || len(page.Months) != 1 || math.Abs(page.Months[0].Gallons-8.4) > 1e-9 {
		t.Fatalf("the tank, the cost and the month: %.400s", body)
	}

	// a 40 mile drive that took the tank from 90% to 70%: 20% of 14 gallons
	start := t0 + 3600
	for i, pct := range []float64{90, 88, 84, 80, 76, 72, 70} {
		level(start+int64(i)*300, pct)
	}
	end := start + 1800
	level(end+30, 80) // someone fills up right after parking: the tank at the end of the drive is still the 70%
	tr := &store.Trip{StartTS: start, Category: "personal", CategorySource: "default"}
	_ = srv.S.CreateTrip(tr)
	tr.EndTS, tr.DistanceKm, tr.Open = &end, 64.4, false
	_ = srv.S.UpdateTrip(tr)
	var d struct {
		Trip struct {
			StartFuel, EndFuel, FuelGal, KmPerTank, EstCost *float64
		}
	}
	_, body = do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d?samples=0", ts.URL, tr.ID), "", nil)
	if json.Unmarshal([]byte(body), &d) != nil || d.Trip.FuelGal == nil || math.Abs(*d.Trip.FuelGal-2.8) > 0.01 || *d.Trip.StartFuel != 90 || *d.Trip.EndFuel != 70 ||
		math.Abs(*d.Trip.KmPerTank-322) > 0.5 || d.Trip.EstCost == nil || math.Abs(*d.Trip.EstCost-2.8*3.25) > 0.01 {
		t.Fatalf("the trip's fuel: %.500s", body)
	}

	// a fill-up in the middle of a drive leaves that drive without a number rather than a wrong one
	level(start+2100, 95)
	tr2 := &store.Trip{StartTS: start + 1900, Category: "personal", CategorySource: "default"}
	_ = srv.S.CreateTrip(tr2)
	end2 := start + 2400
	tr2.EndTS, tr2.DistanceKm, tr2.Open = &end2, 10, false
	_ = srv.S.UpdateTrip(tr2)
	_, body = do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d?samples=0", ts.URL, tr2.ID), "", nil)
	if strings.Contains(body, `"fuelGal":2`) || strings.Contains(body, `"fuelGal":1`) || strings.Contains(body, `"fuelGal":0`) {
		t.Fatalf("a pump stop in the middle is not fuel burned: %.400s", body)
	}
}

func TestAFillUpCanBeAddedByHandDismissedOrDeleted(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if code, _ := do(t, admin, "POST", ts.URL+"/api/fuel/fills", `{"pricePerGal":3.1}`, xh); code != 400 {
		t.Fatal("the gallons are needed")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/fuel/fills", `{"gallons":11,"pricePerGal":3.1,"startPct":20,"endPct":100}`, xh); code != 200 {
		t.Fatal("add by hand")
	}
	list, _ := srv.S.ListRefuels()
	if len(list) != 1 || list[0].Source != "manual" || list[0].Status != "logged" {
		t.Fatalf("kept: %+v", list)
	}
	if g, n := fuel.TankGallons(list); n != 1 || math.Abs(g-13.75) > 0.01 {
		t.Fatalf("a manual fill-up with levels teaches the tank: %v %v", g, n)
	}
	_ = srv.S.SaveRefuel(&fuel.Fill{TS: time.Now().Unix(), StartPct: 10, EndPct: 40, Status: "pending", Source: "auto"})
	list, _ = srv.S.ListRefuels()
	id := list[1].ID
	if code, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/fuel/fills/%d", ts.URL, id), `{"status":"dismissed"}`, xh); code != 200 {
		t.Fatal("dismiss")
	}
	if list, _ = srv.S.ListRefuels(); len(list) != 1 {
		t.Fatalf("a dismissed one is left out: %+v", list)
	}
	if code, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/fuel/fills/%d", ts.URL, list[0].ID), "", xh); code != 200 {
		t.Fatal("delete")
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/fuel", "", nil); code == 200 {
		t.Fatal("signed-out callers get nothing")
	}
}

func TestThePlacesWentToMostAreRanked(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_ = srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Gym", Lat: 44.7, Lon: -85.6, RadiusM: 150})
	_ = srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Cabin", Lat: 45.0, Lon: -85.0, RadiusM: 150})
	t0 := time.Now().Add(-48 * time.Hour).Unix()
	for i, at := range [][2]float64{{44.7, -85.6}, {44.7, -85.6}, {44.7, -85.6}, {45.0, -85.0}} {
		start, end := t0+int64(i)*3600, t0+int64(i)*3600+900
		la, lo := at[0], at[1]
		tr := &store.Trip{StartTS: start, Category: "personal", CategorySource: "default"}
		_ = srv.S.CreateTrip(tr)
		tr.EndTS, tr.DistanceKm, tr.Open, tr.EndLat, tr.EndLon = &end, 10, false, &la, &lo
		_ = srv.S.UpdateTrip(tr)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/analytics/places?days=30", "", nil)
	var d struct {
		Places []struct {
			Name   string
			Visits int
		}
	}
	if json.Unmarshal([]byte(body), &d) != nil || len(d.Places) != 2 || d.Places[0].Name != "Gym" || d.Places[0].Visits != 3 || d.Places[1].Name != "Cabin" {
		t.Fatalf("the most visited first: %s", body)
	}
}
