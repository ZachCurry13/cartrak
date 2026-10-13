package ford

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// fakeFord2 serves FordConnect 2.0 shaped endpoints.
type fakeFord2 struct {
	srv          *httptest.Server
	tokenForms   []url.Values
	slowCalls    int32
	telemetryHit int32
	garageStatus int
	limited      bool
	telemetryErr int // when set, /telemetry answers with this HTTP status and a message
}

const telemetrySample = `{"vehicles":[{"vin":"1FMTK1234ABCD5678","metrics":[
 {"name":"XEV_BATTERY_STATE_OF_CHARGE","value":81.5},
 {"name":"XEV_BATTERY_RANGE","doubleValue":301.0},
 {"name":"IGNITION_STATUS","enumValue":"ACCESSORY"},
 {"name":"ODOMETER","value":12345.6},
 {"name":"POSITION","value":{"latitude":42.3,"longitude":-83.2}},
 {"name":"XEV_PLUG_CHARGER_STATUS","enumValue":"CONNECTED"}]}]}`

func newFake2(t *testing.T) *fakeFord2 {
	f := &fakeFord2{garageStatus: 200}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.tokenForms = append(f.tokenForms, r.PostForm)
		w.Write([]byte(`{"access_token":"ACC","refresh_token":"REF2","expires_in":1200}`))
	})
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer ACC" {
			w.WriteHeader(401)
			return false
		}
		return true
	}
	mux.HandleFunc("/telemetry", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		atomic.AddInt32(&f.telemetryHit, 1)
		if f.telemetryErr != 0 {
			w.WriteHeader(f.telemetryErr)
			w.Write([]byte(`{"message":"Internal error talking to the vehicle service","status":5000,"vin":"1FMTK1234ABCD5678"}`))
			return
		}
		if f.limited {
			w.Header().Set("Retry-After", "42")
			w.WriteHeader(429)
			return
		}
		w.Write([]byte(telemetrySample))
	})
	mux.HandleFunc("/garage", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.slowCalls, 1)
		if f.garageStatus != 200 {
			w.WriteHeader(f.garageStatus)
			return
		}
		w.Write([]byte(`{"vehicles":[{"vin":"1FMTK1234ABCD5678","model":"Mustang Mach-E"}]}`))
	})
	for _, p := range []string{"/vehicle-health/alerts", "/wallbox", "/electric/departure-times", "/electric/charge-schedules"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&f.slowCalls, 1)
			w.Write([]byte(`{"ok":true}`))
		})
	}
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeFord2) client(kv memKV) *Connect2 {
	return NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec", RedirectURL: "https://cartrak.example.com/ford/callback",
			APIBase: f.srv.URL, TokenURL: f.srv.URL + "/token"}
	}, kv)
}

func TestLinkURLAndTokenExchangeV2(t *testing.T) {
	f := newFake2(t)
	kv := memKV{}
	c := f.client(kv)

	u, _ := url.Parse(c.LinkURL("st8"))
	q := u.Query()
	if q.Get("client_id") != "cid" || q.Get("response_type") != "code" || q.Get("state") != "st8" ||
		q.Get("redirect_uri") != "https://cartrak.example.com/ford/callback" {
		t.Fatalf("link parameters wrong: %v", q)
	}
	// Ford's own sign-in start, with only the standard fields (no scope, no response_mode).
	if u.Host != "api.vehicle.ford.com" || u.Path != "/fcon-public/v1/auth/init" || q.Get("scope") != "" || q.Get("response_mode") != "" {
		t.Fatalf("wrong sign-in start address or extra fields: %s", u)
	}

	if err := c.Exchange(context.Background(), "the-code"); err != nil || kv[refreshKey] != "REF2" {
		t.Fatalf("exchange: %v %v", err, kv)
	}
	form := f.tokenForms[0]
	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "the-code" || form.Get("client_secret") != "sec" ||
		form.Get("scope") != "cid offline_access openid" || form.Get("redirect_uri") == "" {
		t.Fatalf("token request wrong: %v", form)
	}
}

func TestFetchParsesMetricList(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{refreshKey: "R0"})

	s, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.tokenForms[0].Get("grant_type") != "refresh_token" || f.tokenForms[0].Get("refresh_token") != "R0" {
		t.Fatalf("should refresh with the stored token: %v", f.tokenForms[0])
	}
	if s.SoCPct == nil || *s.SoCPct != 81.5 || s.RangeKm == nil || *s.RangeKm != 301 || s.OdometerKm == nil || *s.OdometerKm != 12345.6 {
		t.Fatalf("metrics not parsed: soc=%v range=%v odo=%v", s.SoCPct, s.RangeKm, s.OdometerKm)
	}
	if s.Ignition != "ACCESSORY" || s.Plug != "CONNECTED" || s.Lat == nil || *s.Lat != 42.3 || *s.Lon != -83.2 {
		t.Fatalf("status/location not parsed: %+v", s)
	}
}
