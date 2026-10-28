package server

import (
	"strings"
	"testing"
)

// fakeCars is only what tracking asks of the fleet: the servers of the cars, in order.
type fakeCars struct {
	CarManager
	servers []*Server
}

func (f fakeCars) EachCar(fn func(id int64, s *Server)) {
	for i, s := range f.servers {
		fn(int64(i+1), s)
	}
}

func TestTrackingEveryCarAtOnceAndExportingThemInFolders(t *testing.T) {
	ts1, a := healthServer(t, fakeProv{}, false, false)
	_, b := healthServer(t, fakeProv{}, false, false)
	a.Cars = fakeCars{servers: []*Server{a, b}}
	admin := adminClient(t, ts1.URL)

	if code, body := do(t, admin, "POST", ts1.URL+"/api/track/start?all=1", `{"hours":1}`, xh); code != 200 || !a.trackOn() || !b.trackOn() {
		t.Fatalf("both cars are tracking after one start: %d %s a=%v b=%v", code, body, a.trackOn(), b.trackOn())
	}
	do(t, admin, "GET", ts1.URL+"/api/trips", "", nil)
	do(t, admin, "POST", ts1.URL+"/api/track/stop?all=1", "", xh)
	if a.trackOn() || b.trackOn() {
		t.Fatal("both stopped")
	}
	code, body := do(t, admin, "GET", ts1.URL+"/api/track/export?all=1", "", nil)
	if code != 200 {
		t.Fatalf("export: %d", code)
	}
	f := zipFilesOf(t, body)
	for _, name := range []string{"README.txt", "car-1/requests.csv", "car-1/debug-report.txt", "car-2/requests.csv", "car-2/readings-raw.jsonl"} {
		if _, ok := f[name]; !ok {
			t.Fatalf("the all-cars export lacks %s: %v", name, mapKeys(f))
		}
	}
	if !strings.Contains(f["car-1/requests.csv"], "GET /api/trips") {
		t.Fatalf("the car that was on screen has the request: %s", f["car-1/requests.csv"])
	}
	// without all=1 only this car is touched
	do(t, admin, "POST", ts1.URL+"/api/track/start", `{"hours":1}`, xh)
	if !a.trackOn() || b.trackOn() {
		t.Fatalf("only this car starts: a=%v b=%v", a.trackOn(), b.trackOn())
	}
}
