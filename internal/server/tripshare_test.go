package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestASavedTripCanBeSharedAsAPlainPageThatHidesTheStart(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(keyAddress, "cartrak.example.test")
	admin := adminClient(t, ts.URL)
	anon := &http.Client{}
	plan := `{"body":{"dest":{"name":"Traverse City"}},"result":{"currency":"$","days":[{"departTs":1790000000,"startPct":90,"km":300,"driveMin":180,"chargeMin":30,"chargeStops":1,"stops":[
{"name":"Home Street 1","lat":45.1,"lon":-93.4,"kind":"start","arriveMin":0,"departMin":0,"arrivePct":90,"departPct":90},
{"name":"Supercharger: Midway","lat":46.2,"lon":-92.1,"kw":250,"kind":"charge","arriveMin":90,"departMin":120,"arrivePct":40,"departPct":80,"kwh":35,"chargeMin":30,"cost":14.5,"legKm":150},
{"name":"Traverse City","lat":44.76,"lon":-85.6,"kind":"dest","arriveMin":210,"arrivePct":35,"departPct":35,"legKm":150}]}]}}`
	body, _ := json.Marshal(map[string]any{"name": "Up north", "body": json.RawMessage(plan)})
	if c, _ := do(t, admin, "POST", ts.URL+"/api/saved-trips", string(body), xh); c != 200 {
		t.Fatalf("save: %d", c)
	}
	_, list := do(t, admin, "GET", ts.URL+"/api/saved-trips", "", nil)
	var l []struct {
		ID    int64
		Share *struct{ URL string }
	}
	json.Unmarshal([]byte(list), &l)
	if len(l) != 1 || l[0].Share != nil {
		t.Fatalf("not shared yet: %s", list)
	}
	id := "1"
	_, out := do(t, admin, "POST", ts.URL+"/api/saved-trips/"+id+"/share", `{"days":7}`, xh)
	var sh struct{ URL, Path string }
	json.Unmarshal([]byte(out), &sh)
	if !strings.HasPrefix(sh.URL, "https://cartrak.example.test/t/") {
		t.Fatalf("share: %s", out)
	}
	c, page := do(t, anon, "GET", ts.URL+sh.Path, "", nil)
	if c != 200 || !strings.Contains(page, "Up north") || !strings.Contains(page, "Supercharger: Midway") || !strings.Contains(page, "Traverse City") || !strings.Contains(page, "35 kWh") {
		t.Fatalf("the page shows the plan: %d %s", c, page)
	}
	if strings.Contains(page, "Home Street 1") || strings.Contains(page, "45.1") || strings.Contains(page, "<script") {
		t.Fatalf("the start must be hidden by default and the page must have no scripts: %s", page)
	}
	if !strings.Contains(page, "google.com/maps/dir") || strings.Contains(page, "origin=45.1") {
		t.Fatalf("a maps link without the start as its origin: %s", page)
	}
	if c, _ := do(t, anon, "GET", ts.URL+"/t/not-the-secret", "", nil); c != 404 {
		t.Fatalf("a wrong secret gave %d", c)
	}
	_, list = do(t, admin, "GET", ts.URL+"/api/saved-trips", "", nil)
	if !strings.Contains(list, sh.Path) || !strings.Contains(list, `"opens":1`) {
		t.Fatalf("the list shows the link and how often it was opened: %s", list)
	}
	// showing the start is a choice
	_, out = do(t, admin, "POST", ts.URL+"/api/saved-trips/"+id+"/share", `{"days":7,"hideStart":false}`, xh)
	json.Unmarshal([]byte(out), &sh)
	if _, page = do(t, anon, "GET", ts.URL+sh.Path, "", nil); !strings.Contains(page, "Home Street 1") {
		t.Fatalf("with the start shown: %s", page)
	}
	do(t, admin, "DELETE", ts.URL+"/api/saved-trips/"+id+"/share", "", xh)
	if c, _ := do(t, anon, "GET", ts.URL+sh.Path, "", nil); c != 404 {
		t.Fatalf("a stopped link gave %d", c)
	}
}
