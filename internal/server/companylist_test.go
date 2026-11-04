package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestCompanyListFindsRemembersAndAsks(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "Acme Widgets" {
			w.Write([]byte(`[{"lat":"44.9","lon":"-93.2","display_name":"Acme Widgets, 1 Main St, Townsville","name":"Acme Widgets","type":"office"}]`))
			return
		}
		w.Write([]byte(`[{"lat":"45.1","lon":"-93.3","display_name":"Something Else, 9 Oak St, Townsville","name":"Something Else"}]`))
	}))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	admin := adminClient(t, ts.URL)
	type resp struct {
		Rows []struct{ Name, Ref, Status string }
		More int
	}
	ask := func(text string) resp {
		b, _ := json.Marshal(map[string]string{"text": text})
		code, body := do(t, admin, "POST", ts.URL+"/api/company-list/resolve", string(b), xh)
		var r resp
		if code != 200 || json.Unmarshal([]byte(body), &r) != nil {
			t.Fatalf("resolve: %d %s", code, body)
		}
		return r
	}
	r := ask("Acme Widgets-1234567-U.S.A.\nMystery Metals-7654321-U.S.A.")
	if len(r.Rows) != 2 || r.Rows[0].Status != "found" || r.Rows[1].Status != "check" || r.Rows[1].Ref != "7654321" {
		t.Fatalf("first: %+v", r)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/company-list/remember", `{"ref":"7654321","name":"Mystery Metals","address":"9 Oak St","lat":45.1,"lon":-93.3}`, xh); c != 200 {
		t.Fatalf("remember: %d", c)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/company-list/remember", `{"ref":"x1","name":"Bad","lat":1,"lon":1}`, xh); c != 400 {
		t.Fatalf("a non-number ref should be refused, got %d", c)
	}
	if r = ask("Mystery Metals-7654321-U.S.A."); r.Rows[0].Status != "remembered" {
		t.Fatalf("second: %+v", r)
	}
}
