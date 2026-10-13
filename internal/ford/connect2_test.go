package ford

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeFord2 serves FordConnect 2.0 shaped endpoints.
type fakeFord2 struct {
	srv          *httptest.Server
	tokenForms   []url.Values
	slowCalls    int32
	telemetryHit int32
	garageStatus int
	limited      bool
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

func TestFetchParsesMetricListAndCachesSlowEndpoints(t *testing.T) {
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
	if !strings.Contains(string(s.Raw), `"garage"`) || !strings.Contains(string(s.Raw), `"chargeSchedules"`) {
		t.Fatalf("slow endpoints missing from raw: %s", s.Raw)
	}
	if got := atomic.LoadInt32(&f.slowCalls); got != 5 {
		t.Fatalf("expected 5 slow endpoint calls, got %d", got)
	}
	// Second fetch: telemetry again, slow endpoints served from the cache.
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&f.slowCalls); got != 5 || atomic.LoadInt32(&f.telemetryHit) != 2 {
		t.Fatalf("cache not used: slow=%d telemetry=%d", got, f.telemetryHit)
	}
	// After the interval they are read again.
	c.slowAt = time.Now().Add(-2 * slowEvery)
	c.Fetch(context.Background())
	if got := atomic.LoadInt32(&f.slowCalls); got != 10 {
		t.Fatalf("slow endpoints not refreshed: %d", got)
	}
}

func TestSlowEndpointFailureIsNotFatal(t *testing.T) {
	f := newFake2(t)
	f.garageStatus = 403
	c := f.client(memKV{refreshKey: "R0"})
	s, err := c.Fetch(context.Background())
	if err != nil || s.SoCPct == nil {
		t.Fatalf("telemetry should still work: %v", err)
	}
	if !strings.Contains(string(s.Raw), `"_endpointErrors"`) || !strings.Contains(string(s.Raw), "HTTP 403") {
		t.Fatalf("failure not recorded: %s", s.Raw)
	}
}

func TestRateLimitV2(t *testing.T) {
	f := newFake2(t)
	f.limited = true
	c := f.client(memKV{refreshKey: "R0"})
	_, err := c.Fetch(context.Background())
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.After != 42*time.Second {
		t.Fatalf("want RateLimitError(42s), got %v", err)
	}
}

func TestNotLinkedV2(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{})
	if c.Ready() {
		t.Fatal("ready without a token")
	}
	if _, err := c.Fetch(context.Background()); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("want ErrNotLinked, got %v", err)
	}
}

func TestTokenErrorDoesNotLeakBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"AADB2C: the secret sec-123 is wrong"}`))
	}))
	defer srv.Close()
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec-123", TokenURL: srv.URL}
	}, memKV{})
	err := c.Exchange(context.Background(), "x")
	if err == nil || strings.Contains(err.Error(), "sec-123") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("error should show only the short code: %v", err)
	}
}

// Some Ford apps reject the scope field on the token request; the exchange then retries without it.
func TestTokenExchangeRetriesWithoutScopeOn400(t *testing.T) {
	var forms []url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		forms = append(forms, r.PostForm)
		if r.PostForm.Get("scope") != "" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"invalid_scope"}`))
			return
		}
		w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":1200}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	kv := memKV{}
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec", RedirectURL: "https://x.example", TokenURL: srv.URL + "/token"}
	}, kv)
	if err := c.Exchange(context.Background(), "code1"); err != nil || kv[refreshKey] != "R" {
		t.Fatalf("exchange should succeed on the retry: %v %v", err, kv)
	}
	if len(forms) != 2 || forms[0].Get("scope") == "" || forms[1].Get("scope") != "" || forms[1].Get("code") != "code1" {
		t.Fatalf("expected a with-scope try then a no-scope retry: %v", forms)
	}
}

func TestTokenErrorsOtherThan400AreNotRetried(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.WriteHeader(401) }))
	defer srv.Close()
	c := NewConnect2(func() Connect2Opts { return Connect2Opts{ClientID: "c", ClientSecret: "s", TokenURL: srv.URL} }, memKV{})
	if err := c.Exchange(context.Background(), "x"); err == nil || hits != 1 {
		t.Fatalf("a 401 must fail once, not retry: err=%v hits=%d", err, hits)
	}
}

// The real telemetry shape (as read by other FordConnect Query clients):
// vehicles[].metrics.<camelCaseName>.value, with position.value.location.{lat,lon}.
func TestParseRealTelemetryShape(t *testing.T) {
	raw := []byte(`{"telemetry":{"vehicles":[{"vin":"X","metrics":{
	  "xevBatteryStateOfCharge":{"value":72.5},
	  "xevBatteryRange":{"value":260.4},
	  "odometer":{"value":10234.5},
	  "ignitionStatus":{"value":"OFF"},
	  "xevPlugChargerStatus":{"value":"DISCONNECTED"},
	  "xevBatteryChargeDisplayStatus":{"value":"NOT_READY"},
	  "outsideTemperature":{"value":12.5},
	  "batteryVoltage":{"value":12.7},
	  "speed":{"value":10},
	  "doorLockStatus":{"value":"LOCKED"},
	  "position":{"value":{"location":{"lat":42.31,"lon":-83.2,"alt":190}}}
	}}]}}`)
	s := ParseState(raw, time.Now())
	if s.SoCPct == nil || *s.SoCPct != 72.5 || s.RangeKm == nil || *s.RangeKm != 260.4 || s.OdometerKm == nil || *s.OdometerKm != 10234.5 {
		t.Fatalf("battery/odometer not parsed: soc=%v range=%v odo=%v", s.SoCPct, s.RangeKm, s.OdometerKm)
	}
	if s.Lat == nil || *s.Lat != 42.31 || s.Lon == nil || *s.Lon != -83.2 {
		t.Fatalf("location not parsed: %v %v", s.Lat, s.Lon)
	}
	if s.Ignition != "OFF" || s.Plug != "DISCONNECTED" || s.ChargeState != "NOT_READY" {
		t.Fatalf("status not parsed: %+v", s)
	}
	if s.OutsideC == nil || *s.OutsideC != 12.5 || s.Battery12V == nil || *s.Battery12V != 12.7 || s.SpeedKph == nil || *s.SpeedKph != 36 {
		t.Fatalf("temperature/voltage/speed not parsed: %v %v %v", s.OutsideC, s.Battery12V, s.SpeedKph)
	}
	if s.Locked == nil || !*s.Locked {
		t.Fatalf("lock state not parsed: %v", s.Locked)
	}
}
