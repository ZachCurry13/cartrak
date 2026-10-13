package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestServiceRemindersCountFromNowAndCanBeMarkedDone(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	odo := 20000.0
	srv.S.SaveState("t", &ford.State{TS: time.Now(), OdometerKm: &odo, Raw: []byte(`{}`)})

	post := func(body string) (int, string) { return do(t, admin, "POST", ts.URL+"/api/maintenance", body, xh) }
	if c, _ := post(`{"name":"x"}`); c != 400 {
		t.Fatalf("an interval is needed: %d", c)
	}
	if c, _ := post(`{"name":"x","everyKm":5}`); c != 400 {
		t.Fatalf("a silly distance: %d", c)
	}
	code, b := post(`{"name":"Tire rotation","everyKm":10000,"everyDays":365}`)
	var made struct{ ID int64 }
	if code != 200 || json.Unmarshal([]byte(b), &made) != nil || made.ID == 0 {
		t.Fatalf("create: %d %s", code, b)
	}
	list := func() (items []struct {
		Name   string
		State  string
		KmLeft *float64
	}) {
		_, b := do(t, admin, "GET", ts.URL+"/api/maintenance", "", nil)
		var out struct {
			Items []struct {
				Name   string
				State  string
				KmLeft *float64
			}
		}
		json.Unmarshal([]byte(b), &out)
		return out.Items
	}
	if l := list(); len(l) != 1 || l[0].State != "ok" || l[0].KmLeft == nil || *l[0].KmLeft != 10000 {
		t.Fatalf("counting from the odometer now: %+v", l)
	}
	odo2 := 30500.0
	srv.S.SaveState("t", &ford.State{TS: time.Now().Add(time.Minute), OdometerKm: &odo2, Raw: []byte(`{}`)})
	if l := list(); l[0].State != "overdue" {
		t.Fatalf("10,500 km later: %+v", l)
	}
	if c, _ := do(t, admin, "POST", fmt.Sprintf("%s/api/maintenance/%d/done", ts.URL, made.ID), "", xh); c != 200 {
		t.Fatal("done")
	}
	if l := list(); l[0].State != "ok" || *l[0].KmLeft != 10000 {
		t.Fatalf("done starts a new interval: %+v", l)
	}
	if c, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/maintenance/%d", ts.URL, made.ID), `{"name":"Rotate tires","everyKm":8000}`, xh); c != 200 {
		t.Fatal("edit")
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/maintenance", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
	if c, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/maintenance/%d", ts.URL, made.ID), "", xh); c != 200 || len(list()) != 0 {
		t.Fatal("delete")
	}
}
