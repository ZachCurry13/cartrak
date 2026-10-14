package server

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type bandOut struct {
	Source    string   `json:"source"`
	BandKm    *float64 `json:"bandKm"`
	CarBandKm *float64 `json:"carBandKm"`
	KmPerPct  float64  `json:"kmPerPct"`
	Trips     int      `json:"trips"`
	Low, High int
	Months    []struct {
		Month  string  `json:"month"`
		BandKm float64 `json:"bandKm"`
	} `json:"months"`
}

func TestTheRangeBetweenTwoLevelsComesFromRealDrivingAndFallsBackToTheCar(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	get := func(q string) bandOut {
		code, b := do(t, admin, "GET", ts.URL+"/api/range-band"+q, "", nil)
		var o bandOut
		if code != 200 || json.Unmarshal([]byte(b), &o) != nil {
			t.Fatalf("%s: %d %s", q, code, b)
		}
		return o
	}

	if o := get(""); o.BandKm != nil || o.Source != "" || len(o.Months) != 0 {
		t.Fatalf("nothing is known before there is any driving: %+v", o)
	}

	// The car's own figure: 200 km at 50 percent is 4 km per percent.
	soc, rng := 50.0, 200.0
	srv.S.SaveState("t", &ford.State{TS: time.Now(), SoCPct: &soc, RangeKm: &rng, Raw: []byte(`{}`)})
	if o := get(""); o.Source != "car" || o.BandKm == nil || math.Abs(*o.BandKm-240) > 1e-6 {
		t.Fatalf("with no trips the car's estimate for 20 to 80 percent is 240 km: %+v", o)
	}

	// Real driving at 5 km per percent over the last two weeks.
	for i, c := range []struct{ km, from, to float64 }{{100, 90, 70}, {75, 70, 55}, {125, 55, 30}} {
		start := time.Now().Add(-time.Duration(i+2) * 48 * time.Hour)
		end, from, to := start.Add(time.Hour).Unix(), c.from, c.to
		tr := &store.Trip{StartTS: start.Unix(), StartSoC: &from, TZ: "America/New_York"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.EndSoC, tr.DistanceKm, tr.Open = &end, &to, c.km, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	o := get("?low=20&high=80&days=30")
	if o.Source != "trips" || o.Trips != 3 || math.Abs(o.KmPerPct-5) > 1e-6 || o.BandKm == nil || math.Abs(*o.BandKm-300) > 1e-6 {
		t.Fatalf("real driving beats the car's guess: %+v", o)
	}
	if o.CarBandKm == nil || math.Abs(*o.CarBandKm-240) > 1e-6 {
		t.Fatalf("the car's figure is still shown for comparison: %+v", o)
	}
	if len(o.Months) > 0 && o.Months[0].BandKm == 0 { // months need two trips each, so a month boundary may split these
		t.Fatalf("month rows carry a band: %+v", o.Months)
	}
	if o := get("?low=10&high=90&days=30"); o.BandKm == nil || math.Abs(*o.BandKm-400) > 1e-6 || o.Low != 10 || o.High != 90 {
		t.Fatalf("a wider band: %+v", o)
	}
	if o := get("?low=70&high=30"); o.High != 71 {
		t.Fatalf("a backwards band is corrected: %+v", o)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/range-band", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}

func TestTheRangeIsAlsoWorkedOutByWeatherAndForTodaysTemperature(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	mk := func(daysAgo int, temp, kmPerPct float64) {
		start := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
		tc, odo := temp, 100.0
		srv.S.SaveState("t", &ford.State{TS: start.Add(10 * time.Minute), OutsideC: &tc, OdometerKm: &odo, Raw: []byte(`{}`)})
		from, to := 80.0, 60.0 // 20 percent used
		tr := &store.Trip{StartTS: start.Unix(), StartSoC: &from, TZ: "UTC"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		end := start.Add(time.Hour).Unix()
		tr.EndTS, tr.EndSoC, tr.DistanceKm, tr.Open = &end, &to, 20*kmPerPct, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	for i := 3; i <= 5; i++ {
		mk(i, -5, 3) // winter: 3 km for every percent
		mk(i+10, 25, 5)
	}
	now := time.Now()
	cold := -3.0
	srv.S.SaveState("t", &ford.State{TS: now, OutsideC: &cold, Raw: []byte(`{}`)})

	_, body := do(t, admin, "GET", ts.URL+"/api/range-band?low=20&high=80&days=60", "", nil)
	var o struct {
		ByTemp []struct {
			Name   string
			BandKm float64
		}
		OutsideC float64
		NowBand  *struct {
			Name   string
			BandKm float64
		}
	}
	if err := json.Unmarshal([]byte(body), &o); err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, b := range o.ByTemp {
		got[b.Name] = b.BandKm
	}
	if len(got) != 2 || math.Abs(got["Below freezing"]-180) > 1e-6 || math.Abs(got["20 °C and above"]-300) > 1e-6 {
		t.Fatalf("a stretch of battery goes 180 km in the cold and 300 km in the warm: %s", body)
	}
	if o.OutsideC != -3 || o.NowBand == nil || o.NowBand.Name != "Below freezing" || math.Abs(o.NowBand.BandKm-180) > 1e-6 {
		t.Fatalf("for today's temperature: %s", body)
	}
}
