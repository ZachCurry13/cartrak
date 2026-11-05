package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTripChainPicksChargersOthersCanUseAndKeepsTheList(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	hits := 0
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead { // the list is counted once however often its version is asked
			hits++
		}
		var sites []string
		for lat := 45.5; lat < 51; lat += 0.5 {
			sites = append(sites, fmt.Sprintf(`{"name":"Site %.1f","status":"OPEN","gps":{"latitude":%.1f,"longitude":-93.2},"stallCount":8,"powerKilowatt":250,"otherEVs":true,"plugs":{"nacs":8}}`, lat, lat))
			sites = append(sites, fmt.Sprintf(`{"name":"Tesla only %.1f","status":"OPEN","gps":{"latitude":%.1f,"longitude":-93.2},"otherEVs":false,"plugs":{"nacs":8}}`, lat, lat))
		}
		w.Write([]byte("[" + strings.Join(sites, ",") + "]"))
	}))
	defer svc.Close()
	srv.SuperURL = svc.URL

	body := `{"start":{"name":"Home","lat":45,"lon":-93.2},"dest":{"name":"There","lat":50.5,"lon":-93.2},"startPct":90,"reservePct":10,"targetPct":90,"chargeKw":150,"maxLegMin":150,"dayMin":720,"plugs":"nacs"}`
	code, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", body, xh)
	var r struct {
		Days []struct {
			Stops []struct{ Name, Kind string }
		}
		Sites int
	}
	if code != 200 || json.Unmarshal([]byte(out), &r) != nil || len(r.Days) == 0 || r.Sites != 11 {
		t.Fatalf("plan: %d %s", code, out)
	}
	if strings.Contains(out, "Tesla only") {
		t.Fatal("a site other makes cannot use was offered")
	}
	last := r.Days[len(r.Days)-1].Stops
	if last[len(last)-1].Kind != "dest" || r.Days[0].Stops[0].Kind != "start" {
		t.Fatalf("stops: %s", out)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/trip-chain", body, xh); c != 200 || hits != 1 {
		t.Fatalf("second plan %d, list downloaded %d times", c, hits)
	}
	tight := strings.Replace(body, `"maxLegMin":150`, `"maxLegMin":30`, 1)
	if c, _ := do(t, admin, "POST", ts.URL+"/api/trip-chain", tight, xh); c != 422 {
		t.Fatalf("impossible limits gave %d", c)
	}
}
