package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAPublicLinkShowsOnlyWhatWasChosenAndCanBeTurnedOff(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Minute)
	for i := 0; i < 41; i++ { // 4 km north, a reading every minute
		la, lo := 40+float64(i)*0.0009, -75.0
		srv.S.SaveState("t", &ford.State{TS: start.Add(time.Duration(i) * time.Minute), Lat: &la, Lon: &lo, Raw: []byte(`{}`)})
	}
	end := start.Add(40 * time.Minute).Unix()
	a, b := 100.0, 104.0
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 4, StartOdo: &a, EndOdo: &b, Category: "work", TZ: "UTC"}
	if err := srv.S.AddManualTrip(tr); err != nil {
		t.Fatal(err)
	}

	create := func(body string) (int, string, string) {
		code, out := do(t, admin, "POST", ts.URL+"/api/shares", body, xh)
		var r struct{ Path string }
		json.Unmarshal([]byte(out), &r)
		return code, r.Path, out
	}
	if code, _, _ := create(`{"kind":"trip","id":` + fmt.Sprint(tr.ID) + `,"mode":"secret"}`); code != 400 {
		t.Fatalf("a made-up mode: %d", code)
	}
	code, path, _ := create(fmt.Sprintf(`{"kind":"trip","id":%d,"mode":"trimmed","title":"Trip to the lake","units":"mi","expiresDays":7}`, tr.ID))
	if code != 200 || !strings.HasPrefix(path, "/s/") || len(path) != len("/s/")+12 {
		t.Fatalf("create: %d %s", code, path)
	}
	anon := &http.Client{}
	get := func(p string) (int, string) { return do(t, anon, "GET", ts.URL+p, "", nil) }

	code, body := get(path + "/data")
	var d struct {
		Title       string
		DistanceKm  float64
		DurationSec int64
		Routes      [][][2]float64
		Start       *int64
		Origin      string
	}
	if code != 200 || json.Unmarshal([]byte(body), &d) != nil || d.Title != "Trip to the lake" || d.DistanceKm != 4 || len(d.Routes) != 1 {
		t.Fatalf("data: %d %s", code, body)
	}
	if first := d.Routes[0][0]; first[0] < 40.0035 || d.Routes[0][len(d.Routes[0])-1][0] > 40.0325 {
		t.Fatalf("the first and last 400 m are left out: %v ... %v", first, d.Routes[0][len(d.Routes[0])-1])
	}
	if d.Start != nil || d.Origin != "" {
		t.Fatalf("no clock times or city names unless chosen: %s", body)
	}
	for _, leak := range []string{"vin", "TESTVIN", "category", "odometer", `"lat"`} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(leak)) {
			t.Fatalf("%q must not appear: %s", leak, body)
		}
	}

	resp, err := anon.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex") || resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("the page and its headers: %d %v", resp.StatusCode, resp.Header)
	}

	_, statsPath, _ := create(fmt.Sprintf(`{"kind":"trip","id":%d,"mode":"stats","showTime":true}`, tr.ID))
	_, sb := get(statsPath + "/data")
	var sd struct {
		Routes [][][2]float64
		Start  *int64
		MapsOn bool
	}
	json.Unmarshal([]byte(sb), &sd)
	if len(sd.Routes) != 0 || sd.Start == nil || sd.MapsOn {
		t.Fatalf("a stats link has no route, and shows the time because it was chosen: %s", sb)
	}
	if code, _ := get(statsPath + "/tile/12/1197/1553"); code != 404 {
		t.Fatalf("a stats link gets no map tiles: %d", code)
	}

	if code, _ := get("/s/sh_wrong/data"); code != 404 {
		t.Fatalf("a wrong link: %d", code)
	}
	code, list := do(t, admin, "GET", ts.URL+"/api/shares", "", nil)
	var shares []struct {
		ID    int64
		Views int
	}
	json.Unmarshal([]byte(list), &shares)
	if code != 200 || len(shares) != 2 || strings.Contains(list, path) || strings.Contains(list, "sh_") {
		t.Fatalf("the list never shows the secret: %s", list)
	}
	if shares[1].Views != 1 {
		t.Fatalf("the page view was counted: %+v", shares)
	}
	if c, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/shares/%d", ts.URL, shares[1].ID), "", xh); c != 200 {
		t.Fatal("revoke")
	}
	if code, _ := get(path + "/data"); code != 404 {
		t.Fatalf("a revoked link: %d", code)
	}
	if c, _ := do(t, anon, "POST", ts.URL+"/api/shares", `{}`, xh); c != 401 {
		t.Fatalf("anonymous cannot create links: %d", c)
	}
}

func TestAWholeDayCanBeSharedAndOnlyARealDay(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Now().In(loc).AddDate(0, 0, -2)
	noon := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
	for i, km := range []float64{4, 6} {
		start := noon.Add(time.Duration(i) * 2 * time.Hour)
		end := start.Add(20 * time.Minute).Unix()
		a, b := 100.0+float64(i)*10, 100.0+float64(i)*10+km
		if err := srv.S.AddManualTrip(&store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: km, StartOdo: &a, EndOdo: &b, Category: "medical", TZ: loc.String()}); err != nil {
			t.Fatal(err)
		}
	}
	id := day.Year()*10000 + int(day.Month())*100 + day.Day()
	code, out := do(t, admin, "POST", ts.URL+"/api/shares", fmt.Sprintf(`{"kind":"day","id":%d,"mode":"stats","title":"My day"}`, id), xh)
	var r struct{ Path string }
	_ = json.Unmarshal([]byte(out), &r)
	if code != 200 || !strings.HasPrefix(r.Path, "/s/") || len(r.Path) != len("/s/")+12 {
		t.Fatalf("share a day: %d %s", code, out)
	}
	_, data := do(t, &http.Client{}, "GET", ts.URL+r.Path+"/data", "", nil)
	var d struct {
		Kind       string
		Legs       int
		DistanceKm float64
	}
	if json.Unmarshal([]byte(data), &d) != nil || d.Kind != "day" || d.Legs != 2 || d.DistanceKm != 10 {
		t.Fatalf("both trips of the day, 10 km: %s", data)
	}
	if strings.Contains(data, "medical") {
		t.Fatalf("a link must not name the categories: %s", data)
	}
	for _, bad := range []string{"20261340", "19990101", "0"} {
		if code, _ := do(t, admin, "POST", ts.URL+"/api/shares", `{"kind":"day","id":`+bad+`,"mode":"stats"}`, xh); code == 200 {
			t.Fatalf("%s is not a day", bad)
		}
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/shares", `{"kind":"day","id":20200101,"mode":"stats"}`, xh); code != 404 {
		t.Fatalf("a day with no trips: %d", code)
	}
}

func TestALinkShowsTheEfficiencyDetailsOnlyWhenAsked(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	start := time.Now().Add(-5 * time.Hour).Truncate(time.Minute)
	end := start.Add(30 * time.Minute).Unix()
	a, b := 100.0, 130.0
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 30, StartOdo: &a, EndOdo: &b, MaxSpeedKph: 112, Category: "medical", TZ: "UTC"}
	if err := srv.S.AddManualTrip(tr); err != nil {
		t.Fatal(err)
	}
	_, _ = srv.S.DB.Exec(`UPDATE trips SET max_speed_kph=112 WHERE id=?`, tr.ID)
	data := func(extra string) string {
		_, out := do(t, admin, "POST", ts.URL+"/api/shares", fmt.Sprintf(`{"kind":"trip","id":%d,"mode":"stats"%s}`, tr.ID, extra), xh)
		var r struct{ Path string }
		_ = json.Unmarshal([]byte(out), &r)
		_, body := do(t, &http.Client{}, "GET", ts.URL+r.Path+"/data", "", nil)
		return body
	}
	with, without := data(`,"showInsights":true`), data(`,"showInsights":false`)
	if !strings.Contains(with, `"insights"`) || !strings.Contains(with, `"topKph":112`) || !strings.Contains(with, `"avgKph":60`) {
		t.Fatalf("the details are there when asked: %s", with)
	}
	if strings.Contains(without, "insights") || strings.Contains(without, "112") {
		t.Fatalf("and absent when not: %s", without)
	}
	if strings.Contains(with, "medical") {
		t.Fatalf("never the category: %s", with)
	}
}
