package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// addDrive stores a finished trip of km kilometers, starting at.
func addDrive(t *testing.T, srv *Server, at time.Time, km float64) {
	t.Helper()
	fp := func(v float64) *float64 { return &v }
	end := at.Add(30 * time.Minute).Unix()
	tr := &store.Trip{StartTS: at.Unix(), StartLat: fp(40), StartLon: fp(-75), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.StartKwh, tr.EndKwh, tr.Open = &end, fp(40.1), fp(-75.1), km, fp(60), fp(60-km/6), false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
}

func TestInsightsEndpointsAnswer(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	now := time.Now()
	for i := 1; i <= 5; i++ {
		addDrive(t, srv, now.AddDate(0, 0, -i), 20)
	}
	month := now.Format("2006-01")
	for _, path := range []string{"charging", "driving", "records", "recap", "battery", "badges", "everywhere?days=30", "life?days=30", "wrapped?period=" + month, "wrapped?period=" + now.Format("2006")} {
		code, body := do(t, admin, "GET", ts.URL+"/api/analytics/"+path, "", nil)
		if code != 200 || !json.Valid([]byte(body)) {
			t.Fatalf("%s: %d %s", path, code, body)
		}
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/analytics/everywhere?days=30", "", nil)
	if !strings.Contains(body, `"trips":5`) {
		t.Fatalf("five routes: %s", body)
	}
	for i := 0; i < 210; i++ { // more than one page of the trip list: the analysis still sees every trip
		addDrive(t, srv, now.AddDate(0, 0, -40).Add(time.Duration(i)*time.Hour), 2)
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/analytics/driving?days=0", "", nil); !strings.Contains(body, `"trips":215`) {
		t.Fatalf("every trip counted: %.120s", body)
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/analytics/wrapped?period="+now.AddDate(0, 0, -1).Format("2006-01"), "", nil)
	if !strings.Contains(body, `"personalities"`) || !strings.Contains(body, `"flow"`) || !strings.Contains(body, `"ring"`) {
		t.Fatalf("wrapped parts: %s", body)
	}
}

func TestNewBadgesAndTheMonthlyRecapGoToTheInbox(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	now := time.Now()
	addDrive(t, srv, now.AddDate(0, 0, -3), 50)
	ctx := context.Background()
	srv.newBadges(ctx) // the first look notes what is earned, quietly
	if as, _ := srv.S.RecentAlerts(10); len(as) != 0 {
		t.Fatalf("no burst of old news: %+v", as)
	}
	addDrive(t, srv, now.AddDate(0, 0, -2), 170) // a road trip, and past 100 miles in all
	srv.newBadges(ctx)
	as, _ := srv.S.RecentAlerts(10)
	titles := ""
	for _, a := range as {
		titles += a.Title + "|"
	}
	if !strings.Contains(titles, "Road trip") || !strings.Contains(titles, "First 100 miles") {
		t.Fatalf("new badges told: %s", titles)
	}
	srv.newBadges(ctx)
	if again, _ := srv.S.RecentAlerts(10); len(again) != len(as) {
		t.Fatal("told once")
	}
	// the first of a month, after 9: last month's recap, once, when there was driving in it
	loc := srv.defaultLoc()
	first := time.Date(now.In(loc).Year(), now.In(loc).Month(), 1, 10, 0, 0, 0, loc).AddDate(0, 1, 0) // the first of next month: this month had driving
	srv.recapReady(ctx, first.Add(-2*time.Hour))
	srv.recapReady(ctx, first)
	srv.recapReady(ctx, first.Add(time.Hour))
	n := 0
	all, _ := srv.S.RecentAlerts(20)
	for _, a := range all {
		if a.Kind == "monthly_recap" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("one recap notice: %d", n)
	}
}
