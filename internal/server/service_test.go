package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestALongStopAtAServicePlaceIsAskedAboutOnceAndLogsTheWork(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	shop := &store.NamedPlace{Name: "Tire place", Lat: 40.1, Lon: -75.1, RadiusM: 150, ServiceKind: "tire"}
	_ = srv.S.SaveNamedPlace(shop)
	_ = srv.S.SaveMaintenance(&store.Maintenance{Name: "Tire rotation", EveryKm: fp(12000)})
	now := time.Now()
	// every drive ends at the shop: two stops of an hour there, then one of ten minutes, which is not a visit
	addDrive(t, srv, now.Add(-5*time.Hour), 20)                // ends at 40.1,-75.1 after 30 min
	addDrive(t, srv, now.Add(-5*time.Hour+90*time.Minute), 20) // leaves an hour later
	addDrive(t, srv, now.Add(-2*time.Hour), 20)                // back at the shop
	addDrive(t, srv, now.Add(-2*time.Hour+40*time.Minute), 20) // gone ten minutes later
	srv.findServiceVisits(context.Background(), now)
	srv.findServiceVisits(context.Background(), now) // asked once
	vs, _ := srv.S.Visits(0, 0, "pending")
	if len(vs) != 2 || vs[0].PlaceName != "Tire place" || vs[0].End-vs[0].Start != 3600 || vs[1].End-vs[1].Start != 3600 {
		t.Fatalf("two pending visits of an hour: %d", len(vs))
	}
	if as, _ := srv.S.RecentAlerts(5); len(as) != 2 || !strings.Contains(as[0].Title, "Tire place") || as[0].URL != fmt.Sprintf("#/service/%d", vs[0].ID) {
		t.Fatalf("asked: %+v", as)
	}
	body := `{"status":"logged","items":["Tire rotation","Wipers"],"cost":49.5,"notes":"all four","due":[{"item":"Wipers","everyDays":365}]}`
	if code, b := do(t, admin, "PATCH", fmt.Sprintf("%s/api/service/%d", ts.URL, vs[0].ID), body, xh); code != 200 {
		t.Fatalf("log: %d %s", code, b)
	}
	ms, _ := srv.S.ListMaintenance()
	got := map[string]*store.Maintenance{}
	for _, m := range ms {
		got[m.Name] = m
	}
	if got["Tire rotation"].LastTS != vs[0].End || got["Wipers"] == nil || *got["Wipers"].EveryDays != 365 {
		t.Fatalf("reminders reset and created: %+v %+v", got["Tire rotation"], got["Wipers"])
	}
	_, b := do(t, admin, "GET", ts.URL+"/api/service", "", nil)
	var out struct{ Visits []store.ServiceVisit }
	if json.Unmarshal([]byte(b), &out) != nil || len(out.Visits) != 2 || out.Visits[0].Status != "logged" || *out.Visits[0].Cost != 49.5 || len(out.Visits[0].Items) != 2 {
		t.Fatalf("history: %s", b)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/service", `{"placeName":"Somewhere","items":["Cabin filter"],"cost":30}`, xh); code != 200 {
		t.Fatal("log by hand")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/service", `{"items":[]}`, xh); code != 400 {
		t.Fatal("a visit says what was done")
	}
}
