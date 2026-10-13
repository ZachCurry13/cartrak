package server

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func pngBytes() []byte {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	return b.Bytes()
}

// tileServer is a fake tile provider that counts requests and remembers the User-Agent.
func tileServer(t *testing.T, body []byte, status int) (*httptest.Server, *int32, *string) {
	var hits int32
	var ua string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(status)
		w.Write(body)
	}))
	t.Cleanup(up.Close)
	return up, &hits, &ua
}

func mapServer(t *testing.T, upstream string) (*httptest.Server, *Server) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.TileDir = t.TempDir()
	srv.Version = "9.9.9"
	srv.Cfg.Set(settings.MapTileURLKey, upstream+"/{z}/{x}/{y}.png")
	return ts, srv
}

func TestTileIsDownloadedOnceThenServedFromDisk(t *testing.T) {
	up, hits, ua := tileServer(t, pngBytes(), 200)
	ts, srv := mapServer(t, up.URL)
	admin := adminClient(t, ts.URL)

	for i := 0; i < 3; i++ {
		resp, err := admin.Get(ts.URL + "/api/tiles/12/1000/1500.png")
		if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("tile %d: %v %v", i, resp, err)
		}
		resp.Body.Close()
	}
	if *hits != 1 {
		t.Fatalf("the tile server was asked %d times; it should be asked once and the copy reused", *hits)
	}
	if !strings.HasPrefix(*ua, "CarTrak/9.9.9 ") {
		t.Fatalf("tile servers ask for an identifying User-Agent: %q", *ua)
	}
	if _, err := os.Stat(filepath.Join(srv.TileDir, "12", "1000", "1500")); err != nil {
		t.Fatalf("no copy kept on disk: %v", err)
	}
}

func TestTileRequestsAreChecked(t *testing.T) {
	up, hits, _ := tileServer(t, pngBytes(), 200)
	ts, _ := mapServer(t, up.URL)
	admin := adminClient(t, ts.URL)
	code := func(p string) int {
		r, err := admin.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r.StatusCode
	}
	for _, bad := range []string{"/api/tiles/20/1/1", "/api/tiles/3/8/0", "/api/tiles/3/0/8", "/api/tiles/3/-1/0", "/api/tiles/x/1/1", "/api/tiles/3/../../etc", "/api/tiles/3/1/1.svg"} {
		if c := code(bad); c == 200 {
			t.Errorf("%s should be refused, got %d", bad, c)
		}
	}
	if *hits != 0 {
		t.Fatalf("refused requests must never reach the tile server (%d)", *hits)
	}
	if r, _ := (&http.Client{}).Get(ts.URL + "/api/tiles/3/1/1"); r.StatusCode != 401 {
		t.Fatalf("only signed-in people may use the tile proxy: %d", r.StatusCode)
	}
}

func TestOnlyRealImagesAreEverPassedOn(t *testing.T) {
	for name, body := range map[string][]byte{
		"a web page": []byte("<html><script>alert(1)</script></html>"),
		"an svg":     []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"too big":    append(pngBytes(), bytes.Repeat([]byte{0}, tileMaxSize)...),
	} {
		up, _, _ := tileServer(t, body, 200)
		ts, _ := mapServer(t, up.URL)
		r, _ := adminClient(t, ts.URL).Get(ts.URL + "/api/tiles/5/3/3")
		if r.StatusCode != 502 {
			t.Errorf("%s: got %d, want 502", name, r.StatusCode)
		}
		r.Body.Close()
	}
}

func TestAnOldCopyIsServedWhenTheTileServerIsDown(t *testing.T) {
	up, _, _ := tileServer(t, nil, 503)
	ts, srv := mapServer(t, up.URL)
	admin := adminClient(t, ts.URL)
	if r, _ := admin.Get(ts.URL + "/api/tiles/4/2/2"); r.StatusCode != 502 {
		t.Fatalf("no copy and the provider is down: %d", r.StatusCode)
	}
	p := filepath.Join(srv.TileDir, "4", "2", "2")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, pngBytes(), 0o644)
	old := time.Now().Add(-40 * 24 * time.Hour) // past the freshness limit, before the purge age
	os.Chtimes(p, old, old)
	if r, _ := admin.Get(ts.URL + "/api/tiles/4/2/2"); r.StatusCode != 200 {
		t.Fatalf("an old copy should beat a gap in the map: %d", r.StatusCode)
	}
	srv.purgeTiles(time.Now().Add(30 * 24 * time.Hour))
	if _, err := os.Stat(p); err == nil {
		t.Fatal("tiles older than two months should be purged")
	}
}

func TestMapsCanBeSwitchedOff(t *testing.T) {
	up, hits, _ := tileServer(t, pngBytes(), 200)
	ts, srv := mapServer(t, up.URL)
	admin := adminClient(t, ts.URL)
	_, b := do(t, admin, "GET", ts.URL+"/api/map", "", nil)
	if !strings.Contains(b, `"enabled":true`) || !strings.Contains(b, "OpenStreetMap") || !strings.Contains(b, "/api/tiles/{z}/{x}/{y}") {
		t.Fatalf("map config: %s", b)
	}
	srv.Cfg.Set(settings.MapEnabledKey, "off")
	_, b = do(t, admin, "GET", ts.URL+"/api/map", "", nil)
	if !strings.Contains(b, `"enabled":false`) || strings.Contains(b, "tiles") {
		t.Fatalf("switched off: %s", b)
	}
	if r, _ := admin.Get(ts.URL + "/api/tiles/3/1/1"); r.StatusCode != 404 || *hits != 0 {
		t.Fatalf("with maps off nothing is fetched: %d, %d hits", r.StatusCode, *hits)
	}
}

func TestMapSettingsAreValidated(t *testing.T) {
	for _, bad := range []string{"http://tiles.example.com/{z}/{x}/{y}.png", "https://tiles.example.com/{z}/{x}.png", "ftp://x/{z}/{x}/{y}", "tiles"} {
		if settings.Valid(settings.MapTileURLKey, bad) == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
	for _, good := range []string{"", "https://tiles.example.com/{z}/{x}/{y}.png", "http://localhost:8081/{z}/{x}/{y}.png"} {
		if err := settings.Valid(settings.MapTileURLKey, good); err != nil {
			t.Errorf("%q: %v", good, err)
		}
	}
	if settings.Valid(settings.MapEnabledKey, "maybe") == nil || settings.Valid(settings.MapEnabledKey, "off") != nil {
		t.Fatal("maps are on or off")
	}
}
