package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestOnATripAwayTheBaseIsFoundFromTheNightsAndIsNotAStop(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(x float64) *float64 { return &x }
	loc := srv.defaultLoc()
	now := time.Now().In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -2)
	key := day.Format("2006-01-02")
	baseLat, baseLon, awayLat, awayLon := 44.7631, -85.6206, 44.9100, -85.9900
	_ = srv.S.SetPlace(geocode.Key(baseLat, baseLon), "Traverse City")
	_ = srv.S.SetPlace(geocode.Key(awayLat, awayLon), "Glen Arbor Township")
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45.5, Lon: -84.0, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	if err := srv.S.AddTimeOff(&store.TimeOff{From: day.AddDate(0, 0, -3).Format("2006-01-02"), To: day.AddDate(0, 0, 1).Format("2006-01-02"), Label: "Up north"}); err != nil {
		t.Fatal(err)
	}
	soc := 70.0
	for _, at := range []time.Time{day.Add(2 * time.Hour), day.Add(26 * time.Hour)} { // the car sleeps at the base
		if _, err := srv.S.SaveState("t", &ford.State{TS: at, Lat: f(baseLat), Lon: f(baseLon), SoCPct: &soc, Raw: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	out := finishedTrip(t, srv, day.Add(10*time.Hour), 25, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(baseLat), f(baseLon), f(awayLat), f(awayLon)
		tr.StartKwh, tr.EndKwh = f(60), f(55)
	})
	finishedTrip(t, srv, day.Add(14*time.Hour), 25, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(awayLat), f(awayLon), f(baseLat), f(baseLon)
		tr.StartKwh, tr.EndKwh = f(55), f(50)
	})

	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+key, "", nil)
	var v struct {
		Summary struct {
			Stops     int     `json:"stops"`
			CostPerKm float64 `json:"costPerKm"`
		} `json:"summary"`
		Events []struct {
			Kind, Place, FromPlace string
			Base                   bool
		} `json:"events"`
		Trip *struct {
			Label string  `json:"label"`
			Base  string  `json:"base"`
			Km    float64 `json:"km"`
		} `json:"trip"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("%s", body)
	}
	var drives, atBase int
	for _, e := range v.Events {
		switch {
		case e.Kind == "driving" && e.FromPlace == "Base" && e.Place == "Glen Arbor Township":
			drives++
		case e.Kind == "driving" && e.FromPlace == "Glen Arbor Township" && e.Place == "Base":
			drives++
		case e.Kind == "parked" && e.Base:
			atBase++
			if e.Place != "Base · Traverse City" {
				t.Fatalf("time at the base is named: %+v", e)
			}
		}
	}
	if drives != 2 || atBase == 0 {
		t.Fatalf("drives from and back to the base, and time at it: %s", body)
	}
	if v.Summary.Stops != 1 { // only the stop at Glen Arbor; the evening at the base is not a stop
		t.Fatalf("stops: %d in %s", v.Summary.Stops, body)
	}
	if v.Trip == nil || v.Trip.Label != "Up north" || v.Trip.Base != "Traverse City" || v.Trip.Km != 50 {
		t.Fatalf("the trip so far: %+v", v.Trip)
	}

	_, list := do(t, admin, "GET", ts.URL+"/api/trips", "", nil)
	var trips []struct {
		ID         int64  `json:"id"`
		StartPlace string `json:"startPlace"`
	}
	_ = json.Unmarshal([]byte(list), &trips)
	for _, tr := range trips {
		if tr.ID == out.ID && tr.StartPlace != "Base" {
			t.Fatalf("the trips list says the drive left the base: %+v", tr)
		}
	}
}

func TestADayAtHomeInAModeHasNoBase(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(x float64) *float64 { return &x }
	loc := srv.defaultLoc()
	now := time.Now().In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -2)
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45.5, Lon: -84.0, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	if err := srv.S.AddTimeOff(&store.TimeOff{From: day.AddDate(0, 0, -1).Format("2006-01-02"), To: day.AddDate(0, 0, 1).Format("2006-01-02")}); err != nil {
		t.Fatal(err)
	}
	soc := 70.0
	if _, err := srv.S.SaveState("t", &ford.State{TS: day.Add(2 * time.Hour), Lat: f(45.5), Lon: f(-84.0), SoCPct: &soc, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	finishedTrip(t, srv, day.Add(10*time.Hour), 10, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(45.5), f(-84.0), f(45.6), f(-84.1)
	})
	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+day.Format("2006-01-02"), "", nil)
	var v struct {
		Trip *struct{} `json:"trip"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil || v.Trip != nil {
		t.Fatalf("the car slept at home, so this is not a trip away: %s", body)
	}
}
