package push

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

type memKV map[string]string

func (m memKV) GetKV(k string) string   { return m[k] }
func (m memKV) SetKV(k, v string) error { m[k] = v; return nil }

type memSubs struct {
	mu   sync.Mutex
	subs map[int64]*store.PushSub
	last map[int64]string
}

func (m *memSubs) ListPushSubs(int64) ([]*store.PushSub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*store.PushSub
	for _, s := range m.subs {
		out = append(out, s)
	}
	return out, nil
}
func (m *memSubs) MarkPush(id int64, e string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last[id] = e
	return nil
}
func (m *memSubs) DeletePushSubID(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subs, id)
	return nil
}

// device makes a subscription with real keys, the way a browser would, pointing at a fake push service.
func device(t *testing.T, id int64, endpoint string, events ...string) *store.PushSub {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	rand.Read(auth)
	return &store.PushSub{ID: id, Endpoint: endpoint, Events: events,
		P256dh: base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(auth)}
}

type seen struct {
	mu   sync.Mutex
	hits int
	hdr  http.Header
	body []byte
}

func fakeService(t *testing.T, status int) (*httptest.Server, *seen) {
	s := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.hits, s.hdr, s.body = s.hits+1, r.Header.Clone(), b
		s.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, s
}

func service(subs ...*store.PushSub) (*Service, *memSubs, memKV) {
	ms := &memSubs{subs: map[int64]*store.PushSub{}, last: map[int64]string{}}
	for _, s := range subs {
		ms.subs[s.ID] = s
	}
	kv := memKV{}
	return &Service{KV: kv, Subs: ms, AllowAny: true}, ms, kv
}

func TestAlertIsEncryptedAndSignedWithTheServersIdentity(t *testing.T) {
	srv, got := fakeService(t, 201)
	sub := device(t, 1, srv.URL, ChargeComplete)
	svc, subs, kv := service(sub)

	err := svc.SendTo(context.Background(), sub, Alert{Kind: ChargeComplete, Title: "Charging complete", Body: "Battery at 80%", URL: "#/charging"})
	if err != nil {
		t.Fatal(err)
	}
	h := got.hdr
	if h.Get("Content-Encoding") != "aes128gcm" || h.Get("TTL") != "3600" || !strings.HasPrefix(h.Get("Authorization"), "vapid t=") {
		t.Fatalf("headers: %v", h)
	}
	pub, _ := svc.PublicKey()
	if !strings.Contains(h.Get("Authorization"), "k="+pub) {
		t.Fatalf("the VAPID header must name the server's public key: %s", h.Get("Authorization"))
	}
	if len(got.body) < 40 || strings.Contains(string(got.body), "Charging complete") || strings.Contains(string(got.body), "Battery") {
		t.Fatalf("the message must be encrypted, not readable by the push service: %q", got.body)
	}
	if subs.last[1] != "" {
		t.Fatalf("delivery should be recorded as fine: %q", subs.last[1])
	}
	if kv["vapid_private"] == "" || pub != kv["vapid_public"] {
		t.Fatal("keys are made once and kept")
	}
	if p2, _ := svc.PublicKey(); p2 != pub {
		t.Fatal("the public key must not change between calls")
	}
}

func TestNotifyOnlyReachesDevicesThatWantItAndAreNotInQuietHours(t *testing.T) {
	srv, got := fakeService(t, 201)
	wants := device(t, 1, srv.URL, ChargeComplete, PlugIn)
	other := device(t, 2, srv.URL, PlugOut) // did not ask for this alert
	asleep := device(t, 3, srv.URL, ChargeComplete)
	// Quiet hours around right now (wrapping midnight is fine). A fixed all-day window fails for the one
	// minute at its end, which is how this test once failed at 23:59 in CI.
	now := time.Now()
	asleep.Quiet = now.Add(-5*time.Minute).Format("15:04") + "-" + now.Add(5*time.Minute).Format("15:04")
	svc, _, _ := service(wants, other, asleep)
	svc.Notify(context.Background(), Alert{Kind: ChargeComplete, Title: "t", Body: "b"})
	if got.hits != 1 {
		t.Fatalf("one device wants it and is awake, but %d got it", got.hits)
	}
}

func TestADeviceThatIsGoneIsRemoved(t *testing.T) {
	srv, _ := fakeService(t, 410)
	sub := device(t, 7, srv.URL, PlugIn)
	svc, subs, _ := service(sub)
	if err := svc.SendTo(context.Background(), sub, Alert{Kind: PlugIn, Title: "t"}); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatalf("410 should be reported: %v", err)
	}
	if _, still := subs.subs[7]; still {
		t.Fatal("a device the push service says is gone should be deleted")
	}
	srv2, _ := fakeService(t, 503)
	sub2 := device(t, 8, srv2.URL, PlugIn)
	svc2, subs2, _ := service(sub2)
	if err := svc2.SendTo(context.Background(), sub2, Alert{Kind: PlugIn, Title: "t"}); err == nil {
		t.Fatal("a failing push service must be reported")
	}
	if _, still := subs2.subs[8]; !still || !strings.Contains(subs2.last[8], "503") {
		t.Fatalf("a temporary failure keeps the device and records the error: %v", subs2.last)
	}
}

func TestOnlyRealPushServicesAreEverContacted(t *testing.T) {
	for url, want := range map[string]bool{
		"https://fcm.googleapis.com/fcm/send/abc": true, "https://updates.push.services.mozilla.com/wpush/v2/x": true,
		"https://web.push.apple.com/Qx": true, "https://wns2-par02p.notify.windows.com/w/?token=x": true,
		"http://fcm.googleapis.com/x": false, "https://127.0.0.1:8080/x": false, "https://192.168.1.5/x": false,
		"https://evilgoogleapis.com/x": false, "https://googleapis.com.evil.example/x": false, "https://user:pw@fcm.googleapis.com/x": false, "": false,
	} {
		if AllowedEndpoint(url) != want {
			t.Errorf("%q: want %v", url, want)
		}
	}
	svc, _, _ := service()
	svc.AllowAny = false
	if err := svc.SendTo(context.Background(), device(t, 1, "https://192.168.1.5/internal"), Alert{}); err == nil {
		t.Fatal("an internal address must be refused before any request is made")
	}
}

func TestQuietHours(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 13, h, m, 0, 0, time.UTC) }
	for _, c := range []struct {
		quiet string
		now   time.Time
		want  bool
	}{
		{"", at(3, 0), false}, {"22:00-07:00", at(23, 0), true}, {"22:00-07:00", at(6, 59), true}, {"22:00-07:00", at(7, 0), false},
		{"22:00-07:00", at(12, 0), false}, {"09:00-17:00", at(12, 0), true}, {"09:00-17:00", at(17, 0), false},
		{"10:00-10:00", at(10, 0), false}, {"junk", at(3, 0), false}, {"25:00-07:00", at(3, 0), false},
	} {
		if InQuiet(c.quiet, c.now) != c.want {
			t.Errorf("%q at %v: want %v", c.quiet, c.now.Format("15:04"), c.want)
		}
	}
	if !ValidQuiet("") || !ValidQuiet("22:00-07:00") || ValidQuiet("22:00") || ValidQuiet("aa:bb-cc:dd") || ValidQuiet("24:00-01:00") {
		t.Fatal("quiet hours validation")
	}
}

func TestEveryKindIsKnown(t *testing.T) {
	for _, k := range Kinds {
		if !ValidKind(k.ID) || k.Label == "" || k.Help == "" {
			t.Errorf("incomplete kind %+v", k)
		}
	}
	if ValidKind("nonsense") || len(Kinds) != 24 {
		t.Fatalf("kinds: %d", len(Kinds))
	}
}

func TestEveryAlertIsReportedWithWhoItReached(t *testing.T) {
	srv, _ := fakeService(t, 201)
	wants := device(t, 1, srv.URL, ChargeComplete)
	other := device(t, 2, srv.URL, PlugOut)
	asleep := device(t, 3, srv.URL, ChargeComplete)
	now := time.Now()
	asleep.Quiet = now.Add(-5*time.Minute).Format("15:04") + "-" + now.Add(5*time.Minute).Format("15:04")
	svc, _, _ := service(wants, other, asleep)
	var devices, wanted, quiet, sent int
	svc.Log = func(a Alert, d, w, q, s int) { devices, wanted, quiet, sent = d, w, q, s }
	svc.Notify(context.Background(), Alert{Kind: ChargeComplete, Title: "t", Body: "b"})
	if devices != 3 || wanted != 1 || quiet != 1 || sent != 1 {
		t.Fatalf("3 devices, 1 wanted it and is awake, 1 in quiet hours, 1 accepted: %d %d %d %d", devices, wanted, quiet, sent)
	}
	svc.Notify(context.Background(), Alert{Kind: LowTire, Title: "t"})
	if wanted != 0 || sent != 0 {
		t.Fatalf("nobody asked for this one: %d %d", wanted, sent)
	}
}

func TestNotifyTellsOnlyThePeopleWhoMaySeeTheCarAndNamesTheCar(t *testing.T) {
	srv, got := fakeService(t, 201)
	mine := device(t, 1, srv.URL, ChargeComplete)
	mine.UserID = 7
	theirs := device(t, 2, srv.URL, ChargeComplete)
	theirs.UserID = 8
	svc, _, _ := service(mine, theirs)
	svc.Visible = func(uid int64) bool { return uid == 7 }
	svc.CarName = func() string { return "Truck" }
	var logged int
	svc.Log = func(a Alert, devices, wanted, quiet, sent int) {
		logged++
		if a.Title != "Charging complete" || wanted != 1 || sent != 1 {
			t.Errorf("the log keeps the plain title and counts only those who may see the car: %+v wanted=%d sent=%d", a, wanted, sent)
		}
	}
	svc.Notify(context.Background(), Alert{Kind: ChargeComplete, Title: "Charging complete", Body: "80%"})
	if got.hits != 1 || logged != 1 {
		t.Fatalf("only one person may see this car: %d sends, %d logs", got.hits, logged)
	}
}
