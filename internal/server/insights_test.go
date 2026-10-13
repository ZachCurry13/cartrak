package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestInsightsGroupEfficiencyByMonthAndTemperature(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	base := time.Now().Add(-10 * 24 * time.Hour)
	for i := 0; i < 6; i++ {
		start := base.Add(time.Duration(i) * 24 * time.Hour)
		temp, odo := 5.0, 100.0+float64(i)*100
		srv.S.SaveState("t", &ford.State{TS: start.Add(10 * time.Minute), OutsideC: &temp, OdometerKm: &odo, Raw: []byte(`{}`)})
		end, a, b := start.Add(time.Hour).Unix(), 80.0, 70.0
		tr := &store.Trip{StartTS: start.Unix(), StartKwh: &a, Category: "work", TZ: "UTC"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.EndKwh, tr.DistanceKm, tr.Open = &end, &b, 50, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	code, body := do(t, admin, "GET", ts.URL+"/api/insights", "", nil)
	var rep struct {
		Trips  int
		Months []struct{ KmPerKwh float64 }
		Bands  []struct {
			Name     string
			KmPerKwh float64
		}
		Best []struct{ ID int64 }
	}
	if code != 200 || json.Unmarshal([]byte(body), &rep) != nil {
		t.Fatalf("%d %s", code, body)
	}
	if rep.Trips != 6 || len(rep.Months) == 0 || rep.Months[0].KmPerKwh != 5 || len(rep.Bands) != 1 || rep.Bands[0].Name != "0 to 10 °C" || len(rep.Best) != 3 {
		t.Fatalf("%s", body)
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/insights", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}
