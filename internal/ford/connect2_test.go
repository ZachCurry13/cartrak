package ford

import (
	"context"
	"encoding/json"
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
		q.Get("scope") != "cid offline_access openid" || q.Get("redirect_uri") != "https://cartrak.example.com/ford/callback" {
		t.Fatalf("link parameters wrong: %v", q)
	}
	if !strings.Contains(u.String(), "B2C_1A_FCON_AUTHORIZE") {
		t.Fatalf("policy missing from sign-in address: %s", u)
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

func TestProbeRedactsLocationAndVIN(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{refreshKey: "R0"})
	res := c.Probe(context.Background())
	b, _ := json.Marshal(res)
	s := string(b)
	for _, leak := range []string{"1FMTK1234ABCD5678", "42.3", "-83.2"} {
		if strings.Contains(s, leak) {
			t.Fatalf("probe leaked %q: %s", leak, s)
		}
	}
	if !strings.Contains(s, "81.5") || !strings.Contains(s, "XEV_BATTERY_STATE_OF_CHARGE") {
		t.Fatalf("probe should keep useful values: %s", s)
	}
	if res["telemetry"].Status != 200 || len(res) != 6 {
		t.Fatalf("probe coverage: %d endpoints, telemetry=%d", len(res), res["telemetry"].Status)
	}
}

func TestRedactTokens(t *testing.T) {
	got := tokens("vehicleLongitude_valueGPS")
	want := []string{"vehicle", "longitude", "value", "gps"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v", got)
	}
	if !sensitive("latitude") || !sensitive("positionValue") || sensitive("chargePlatform") || sensitive("batteryTemperature") {
		t.Fatal("sensitivity rules wrong")
	}
}
