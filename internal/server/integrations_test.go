package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

type integOut struct {
	Items []struct {
		ID, State, Note string
		Testable        bool
		Last            *struct {
			OK  bool
			Msg string
		} `json:"lastTest"`
	}
	Attention int
}

func TestIntegrationsListEveryServiceAndTestAKey(t *testing.T) {
	status := 200
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"fuel_stations":[]}`))
	}))
	defer fake.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.AFDCURL = fake.URL
	admin := adminClient(t, ts.URL)
	get := func() integOut {
		_, b := do(t, admin, "GET", ts.URL+"/api/integrations", "", nil)
		var o integOut
		if json.Unmarshal([]byte(b), &o) != nil || len(o.Items) < 10 {
			t.Fatalf("list: %s", b)
		}
		return o
	}
	find := func(o integOut, id string) int {
		for i, x := range o.Items {
			if x.ID == id {
				return i
			}
		}
		t.Fatalf("no %s card", id)
		return -1
	}
	o := get()
	if o.Items[find(o, "ea")].State != "off" || o.Items[find(o, "ea")].Testable || o.Items[find(o, "weather")].State != "off" {
		t.Fatalf("nothing is on yet: %+v", o.Items)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/integrations/test", `{"id":"ea"}`, xh); c != 400 {
		t.Fatalf("testing without a key gave %d", c)
	}
	srv.Cfg.Set(settings.AFDCKeyKey, "test-key")
	if c, b := do(t, admin, "POST", ts.URL+"/api/integrations/test", `{"id":"ea"}`, xh); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatalf("a working key: %d %s", c, b)
	}
	o = get()
	if i := find(o, "ea"); o.Items[i].State != "on" || !o.Items[i].Testable || o.Items[i].Last == nil || !o.Items[i].Last.OK {
		t.Fatalf("the card shows the key and the test: %+v", o.Items[i])
	}
	status = 403
	do(t, admin, "POST", ts.URL+"/api/integrations/test", `{"id":"ea"}`, xh)
	o = get()
	if i := find(o, "ea"); o.Items[i].Last == nil || o.Items[i].Last.OK || o.Attention < 1 {
		t.Fatalf("a failing test is what needs attention: %+v attention=%d", o.Items[i], o.Attention)
	}
	if strings.Contains(string(mustJSON(t, o)), "test-key") {
		t.Fatal("the key must never be in the list")
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/integrations/test", `{"id":"nope"}`, xh); c != 400 {
		t.Fatalf("an unknown service gave %d", c)
	}
	if _, b := do(t, admin, "GET", ts.URL+"/api/integrations/attention", "", nil); !strings.Contains(b, `"attention":`) {
		t.Fatalf("attention: %s", b)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
