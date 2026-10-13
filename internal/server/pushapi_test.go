package server

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/push"
)

// pushServer is a server with Web Push on, delivering to a fake push service that counts messages.
func pushServer(t *testing.T) (*httptest.Server, *Server, *int32, string) {
	var hits int32
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(201)
	}))
	t.Cleanup(svc.Close)
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Push = &push.Service{KV: srv.S, Subs: srv.S, AllowAny: true}
	return ts, srv, &hits, svc.URL
}

// subscription is what a browser's pushManager.subscribe() hands over.
func subscription(endpoint string) string {
	k, _ := ecdh.P256().GenerateKey(rand.Reader)
	auth := make([]byte, 16)
	rand.Read(auth)
	return `{"endpoint":"` + endpoint + `","keys":{"p256dh":"` + base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()) +
		`","auth":"` + base64.RawURLEncoding.EncodeToString(auth) + `"},"label":"My phone"`
}

func TestAPersonAddsADeviceAndGetsATestAlert(t *testing.T) {
	ts, _, hits, svc := pushServer(t)
	admin := adminClient(t, ts.URL)

	code, body := do(t, admin, "GET", ts.URL+"/api/push/config", "", nil)
	var cfg struct {
		PublicKey string `json:"publicKey"`
		Kinds     []struct{ ID string }
		Devices   []map[string]any
	}
	json.Unmarshal([]byte(body), &cfg)
	if code != 200 || len(cfg.PublicKey) < 80 || len(cfg.Kinds) != 10 || len(cfg.Devices) != 0 {
		t.Fatalf("config: %d %s", code, body)
	}
	if strings.Contains(body, "vapid_private") {
		t.Fatal("the private key must never be sent to the browser")
	}

	sub := subscription(svc+"/push/abc") + `,"events":["charge_complete","plug_out"],"quiet":"22:00-07:00"}`
	code, body = do(t, admin, "POST", ts.URL+"/api/push/subscribe", sub, xh)
	if code != 200 || !strings.Contains(body, `"label":"My phone"`) || !strings.Contains(body, `"quiet":"22:00-07:00"`) || strings.Contains(body, "p256dh") {
		t.Fatalf("subscribe: %d %s", code, body)
	}
	if code, body := do(t, admin, "POST", ts.URL+"/api/push/test", `{"endpoint":"`+svc+`/push/abc"}`, xh); code != 200 || *hits != 1 {
		t.Fatalf("test alert: %d %s (%d delivered)", code, body, *hits)
	}
	_, body = do(t, admin, "POST", ts.URL+"/api/push/prefs", `{"endpoint":"`+svc+`/push/abc","events":["low_tire"],"quiet":""}`, xh)
	if !strings.Contains(body, `"events":["low_tire"]`) {
		t.Fatalf("prefs: %s", body)
	}
	_, body = do(t, admin, "POST", ts.URL+"/api/push/unsubscribe", `{"endpoint":"`+svc+`/push/abc"}`, xh)
	if !strings.Contains(body, `"devices":[]`) {
		t.Fatalf("unsubscribe: %s", body)
	}
}

func TestDefaultsAndLimitsAndBadInput(t *testing.T) {
	ts, _, _, svc := pushServer(t)
	admin := adminClient(t, ts.URL)
	_, body := do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription(svc+"/p/1")+`}`, xh)
	if !strings.Contains(body, `"events":["charge_complete","low_tire","low_12v"]`) {
		t.Fatalf("a new device starts with the core alerts: %s", body)
	}
	for name, in := range map[string]string{
		"unknown alert":   subscription(svc+"/p/2") + `,"events":["launch_missiles"]}`,
		"bad quiet hours": subscription(svc+"/p/3") + `,"quiet":"night"}`,
		"bad key":         `{"endpoint":"` + svc + `/p/4","keys":{"p256dh":"AAAA","auth":"AAAA"}}`,
		"not json":        `nope`,
	} {
		if code, _ := do(t, admin, "POST", ts.URL+"/api/push/subscribe", in, xh); code != 400 {
			t.Errorf("%s should be refused, got %d", name, code)
		}
	}
	for i := 0; i < maxDevicesPerPerson; i++ {
		do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription(svc+"/limit/"+string(rune('a'+i)))+`}`, xh)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription(svc+"/limit/overflow")+`}`, xh); code != 400 {
		t.Fatalf("too many devices: %d", code)
	}
}

func TestNobodyCanPointAlertsAtAnInternalAddress(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Push = &push.Service{KV: srv.S, Subs: srv.S} // the real rules: only push services' domains
	admin := adminClient(t, ts.URL)
	for _, ep := range []string{"https://192.168.1.10/hook", "https://localhost:8080/x", "http://fcm.googleapis.com/fcm/send/x", "https://evil.example.com/fcm"} {
		if code, _ := do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription(ep)+`}`, xh); code != 400 {
			t.Errorf("%s should be refused, got %d", ep, code)
		}
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription("https://fcm.googleapis.com/fcm/send/real")+`}`, xh); code != 200 {
		t.Fatalf("a real push service address is fine: %d", code)
	}
}

func TestDevicesBelongToTheirOwner(t *testing.T) {
	ts, _, _, svc := pushServer(t)
	admin := adminClient(t, ts.URL)
	do(t, admin, "POST", ts.URL+"/api/push/subscribe", subscription(svc+"/p/mine")+`}`, xh)
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)

	_, body := do(t, v, "GET", ts.URL+"/api/push/config", "", nil)
	if !strings.Contains(body, `"devices":[]`) {
		t.Fatalf("a viewer must not see someone else's devices: %s", body)
	}
	if code, _ := do(t, v, "POST", ts.URL+"/api/push/test", `{"endpoint":"`+svc+`/p/mine"}`, xh); code != 404 {
		t.Fatalf("a viewer must not send to someone else's device: %d", code)
	}
	do(t, v, "POST", ts.URL+"/api/push/unsubscribe", `{"endpoint":"`+svc+`/p/mine"}`, xh)
	do(t, v, "POST", ts.URL+"/api/push/prefs", `{"endpoint":"`+svc+`/p/mine","events":["trip_end"]}`, xh)
	_, mine := do(t, admin, "GET", ts.URL+"/api/push/config", "", nil)
	if !strings.Contains(mine, "/p/mine") || !strings.Contains(mine, `"events":["charge_complete","low_tire","low_12v"]`) {
		t.Fatalf("someone else's changes must not touch the device: %s", mine)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/push/config", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}
