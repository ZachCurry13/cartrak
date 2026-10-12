package ford

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type memKV map[string]string

func (m memKV) GetKV(k string) string   { return m[k] }
func (m memKV) SetKV(k, v string) error { m[k] = v; return nil }

func TestConnectFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") == "authorization_code" && r.Form.Get("code") != "abc" {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"access_token":"A1","refresh_token":"R2","expires_in":1200}`))
	})
	mux.HandleFunc("/api/fordconnect/vehicles/v1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"vehicles":[{"vehicleId":"v9","vehicleAuthorizationIndicator":1}]}`))
	})
	mux.HandleFunc("/api/fordconnect/vehicles/v1/v9", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer A1" || r.Header.Get("Application-Id") != "app" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"vehicleDetails":{"odometer":1234.5,"batteryChargeLevel":{"value":81,"distanceToEmpty":300}},
		 "vehicleStatus":{"ignitionStatus":{"value":"OFF"},"plugStatus":{"value":true},"chargingStatus":{"value":"ChargingAC"},"lockStatus":{"value":"LOCKED"}}}`))
	})
	mux.HandleFunc("/api/fordconnect/vehicles/v1/v9/location", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"vehicleLocation":{"latitude":"42.3","longitude":"-83.2","speed":0}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	kv := memKV{}
	c := NewConnect(ConnectOpts{ClientID: "id", ClientSecret: "s", ApplicationID: "app",
		RedirectURL: "https://x/cb", APIBase: srv.URL, TokenURL: srv.URL + "/token"}, kv)
	if c.Ready() {
		t.Fatal("ready before link")
	}
	if _, err := c.Fetch(context.Background()); err != ErrNotLinked {
		t.Fatalf("want ErrNotLinked, got %v", err)
	}
	if err := c.Exchange(context.Background(), "abc"); err != nil || kv["ford_refresh"] != "R2" {
		t.Fatalf("exchange: %v %v", err, kv)
	}
	s, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if *s.OdometerKm != 1234.5 || *s.SoCPct != 81 || *s.RangeKm != 300 || *s.Lat != 42.3 || *s.Lon != -83.2 {
		t.Fatalf("bad parse: %+v", s)
	}
	if s.Ignition != "OFF" || s.Plug != "Connected" || s.ChargeState != "ChargingAC" || s.Locked == nil || !*s.Locked {
		t.Fatalf("bad status parse: %+v", s)
	}
}

func TestParseDictionaryNames(t *testing.T) {
	s := ParseState([]byte(`{"XEV_BATTERY_STATE_OF_CHARGE":{"value":55.5},"SPEED":{"value":10},"AMBIENT_TEMP":{"value":21}}`), time.Now())
	if *s.SoCPct != 55.5 || *s.SpeedKph != 36 || *s.OutsideC != 21 {
		t.Fatalf("%+v", s)
	}
}

func TestOptionsApplyWithoutRestart(t *testing.T) {
	var gotApp string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/fordconnect/vehicles/v1", func(w http.ResponseWriter, r *http.Request) {
		gotApp = r.Header.Get("Application-Id")
		w.Write([]byte(`{"vehicles":[{"vehicleId":"v1","vehicleAuthorizationIndicator":1}]}`))
	})
	mux.HandleFunc("/api/fordconnect/vehicles/v1/v1", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":1200}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	o := ConnectOpts{ClientID: "i", ClientSecret: "s", ApplicationID: "first", APIBase: srv.URL, TokenURL: srv.URL + "/token"}
	c := NewConnectFn(func() ConnectOpts { return o }, memKV{"ford_refresh": "R0"})
	if _, err := c.Fetch(context.Background()); err != nil || gotApp != "first" {
		t.Fatalf("first: %v %q", err, gotApp)
	}
	o.ApplicationID, o.VehicleID = "second", "v1" // changed in the admin panel
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	c2 := NewConnectFn(func() ConnectOpts { return o }, memKV{"ford_refresh": "R0"})
	c2.vid = ""
	if _, err := c2.vehicleID(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := c.opts().ApplicationID; got != "second" {
		t.Fatalf("opts not re-read: %q", got)
	}
}
