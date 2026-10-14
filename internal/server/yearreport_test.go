package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestYearSummaryTotalsTheYearsTripsAndRejectsABadYear(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	start := time.Date(time.Now().In(loc).Year(), 1, 5, 9, 0, 0, 0, loc)
	mk := func(at time.Time, km float64, cat string) {
		end := at.Add(30 * time.Minute).Unix()
		tr := &store.Trip{StartTS: at.Unix(), EndTS: &end, DistanceKm: km, Category: cat, CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open = &end, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	mk(start, 40, "work")
	mk(start.AddDate(0, 0, 1), 10, "personal")
	code, b := do(t, admin, "GET", ts.URL+"/api/report/year", "", nil)
	var out struct {
		Year    int
		Years   []int
		Driving struct {
			Trips      int
			Km         float64
			WorkKm     float64
			PersonalKm float64
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	if out.Driving.Trips != 2 || out.Driving.Km != 50 || out.Driving.WorkKm != 40 || out.Driving.PersonalKm != 10 || len(out.Years) != 1 || out.Year != start.Year() {
		t.Fatalf("%+v", out)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/report/year?year=1999", "", nil); code != 400 {
		t.Fatalf("bad year = %d", code)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/report/year?year=2000", "", nil); code != 200 {
		t.Fatal("an empty past year is fine")
	}
}
